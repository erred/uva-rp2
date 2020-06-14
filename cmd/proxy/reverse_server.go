package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"strconv"
	"sync"

	quic "github.com/lucas-clemente/quic-go"
	"github.com/txthinking/socks5"
)

type ReverseServer struct {
	Proxy
}

func NewReverseServer(p *Proxy) *ReverseServer {
	return &ReverseServer{
		Proxy: *p,
	}
}

func (r *ReverseServer) Run(ctx context.Context) {
	r.handleUDP()
}

func (r *ReverseServer) handleUDP() {
	tlsConf, err := generateTLSConfig()
	if err != nil {
		log.Printf("revserse-server: handleUDP gentls: %v", err)
		return
	}

	qListener, err := quic.ListenAddr(r.Proxy.reverseAddress, tlsConf, nil)
	if err != nil {
		log.Printf("revserse-server: handleUDP listen: %v", err)
		return
	}
	defer qListener.Close()

	fmt.Printf("Listening on %s/%s\n",
		qListener.Addr().Network(), qListener.Addr().String(),
	)

	for {
		qSession, err := qListener.Accept(context.Background())
		if err != nil {
			log.Printf("revserse-server: handleUDP accept: %v", err)
			return
		}

		fmt.Printf("Accepted session %s/%s -> %s/%s\n",
			qSession.RemoteAddr().Network(), qSession.RemoteAddr().String(),
			qSession.LocalAddr().Network(), qSession.LocalAddr().String(),
		)

		go r.serveQUIC(qSession)
	}
}

func (r *ReverseServer) serveQUIC(s quic.Session) {
	defer s.CloseWithError(0, "")

	go func() {
		infoStream, err := s.AcceptStream(context.Background())
		if err != nil {
			log.Printf("revserse-server: serveQUIC accept: %v", err)
			return
		}
		defer infoStream.Close()
		b, err := ioutil.ReadAll(infoStream)
		if err != nil {
			log.Printf("revserse-server: serveQUIC readall: %v", err)
			return
		}
		fmt.Printf("Session %s/%s message: %q\n",
			s.RemoteAddr().Network(), s.RemoteAddr().String(),
			string(b),
		)
	}()

	listenAddr := net.JoinHostPort("0.0.0.0", strconv.Itoa(r.localPort))
	r.localPort++

	sServer, err := socksServer(listenAddr)
	if err != nil {
		log.Printf("revserse-server: serveQUIC socks: %v", err)
		return
	}

	fmt.Printf("Listening on tcp/%s udp/%s for session %s/%s\n",
		sServer.TCPAddr.String(), sServer.UDPAddr.String(),
		s.RemoteAddr().Network(), s.RemoteAddr().String(),
	)

	err = sServer.ListenAndServe(&reverseConn{
		sess: s,
		udp:  make(map[string]quic.Stream),
		ss:   sServer,
	})
	if err != nil {
		log.Printf("revserse-server: serveQUIC serve: %v", err)
		return
	}
}

type reverseConn struct {
	sess quic.Session

	// udp
	ul  sync.Mutex
	udp map[string]quic.Stream
	ss  *socks5.Server
}

func (rc *reverseConn) TCPHandle(s *socks5.Server, source *net.TCPConn, r *socks5.Request) error {
	switch r.Cmd {
	case socks5.CmdConnect:
		// open stream to reverse client
		stream, err := rc.sess.OpenStream()
		if err != nil {
			return err
		}
		defer stream.Close()

		// tell reverse client where this connection should go
		err = writeMessage(stream, []byte("tcp"))
		if err != nil {
			return err
		}
		err = writeMessage(stream, []byte(r.Address()))
		if err != nil {
			return err
		}

		// tell socks client this connection is ok to use
		a, addr, port, err := socks5.ParseAddress(source.RemoteAddr().String())
		if err != nil {
			return err
		}
		reply := socks5.NewReply(socks5.RepSuccess, a, addr, port)
		_, err = reply.WriteTo(source)
		if err != nil {
			return err
		}

		// copy data between streams
		return copyConn(stream, source)

	case socks5.CmdUDP:
		caddr, err := r.UDP(source, s.ServerAddr)
		if err != nil {
			return err
		}
		if err := s.TCPWaitsForUDP(caddr); err != nil {
			return err
		}
		return nil
	default:
		return socks5.ErrUnsupportCmd
	}
}

func (rc *reverseConn) UDPHandle(s *socks5.Server, source *net.UDPAddr, d *socks5.Datagram) error {
	var err error
	rc.ul.Lock()
	stream, ok := rc.udp[d.Address()]
	rc.ul.Unlock()

	if !ok {
		stream, err = rc.sess.OpenStream()
		if err != nil {
			return err
		}
		err = writeMessage(stream, []byte("udp"))
		if err != nil {
			return err
		}
		err = writeMessage(stream, []byte(d.Address()))
		if err != nil {
			return err
		}
		rc.ul.Lock()
		rc.udp[d.Address()] = stream
		rc.ul.Unlock()

		go rc.handleIncoming(source, stream)
	}

	return writeMessage(stream, d.Data)
}

func (rc *reverseConn) handleIncoming(local *net.UDPAddr, s quic.Stream) {
	defer s.Close()

	a, addr, port, err := socks5.ParseAddress(local.String())
	if err != nil {
		log.Printf("revserse-server: handleIncoming parse: %v", err)
		return
	}
	for {
		b, err := readMessage(s)
		if err != nil {
			log.Printf("revserse-server: handleIncoming read: %v", err)
			return
		}
		d := socks5.NewDatagram(a, addr, port, b)
		_, err = rc.ss.UDPConn.WriteToUDP(d.Bytes(), local)
		if err != nil {
			log.Printf("revserse-server: handleIncoming write: %v", err)
			return
		}
	}
}

func generateTLSConfig() (*tls.Config, error) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
	}

	// public key
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, key.Public(), key)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	// private key
	priv, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: priv})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-reverse"},
	}, nil
}

// func generateTLSConfig() (*tls.Config, error) {
// 	return generateTLSConfig2(), nil
// }
// func generateTLSConfig2() *tls.Config {
// 	key, err := rsa.GenerateKey(rand.Reader, 1024)
// 	if err != nil {
// 		panic(err)
// 	}
// 	template := x509.Certificate{SerialNumber: big.NewInt(1)}
// 	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
// 	if err != nil {
// 		panic(err)
// 	}
// 	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
// 	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
//
// 	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return &tls.Config{
// 		Certificates: []tls.Certificate{tlsCert},
// 		NextProtos:   []string{"quic-echo-example"},
// 	}
// }

func readMessage(r io.Reader) ([]byte, error) {
	buf := make([]byte, 4)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	buf = make([]byte, binary.BigEndian.Uint32(buf))
	_, err = io.ReadFull(r, buf)
	return buf, err
}

func writeMessage(w io.Writer, b []byte) error {
	l := uint32(len(b))
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, l)
	_, err := w.Write(buf)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
