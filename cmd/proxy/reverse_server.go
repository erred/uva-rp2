package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
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

	lpmu sync.Mutex
}

func NewReverseServer(p *Proxy) *ReverseServer {
	return &ReverseServer{
		Proxy: *p,
	}
}

func (r *ReverseServer) Run(ctx context.Context) {
	tlsConf, err := generateTLSConfig()
	if err != nil {
		log.Printf("reverse-server: handleUDP gentls: %v", err)
		return
	}
	qConf := &quic.Config{
		KeepAlive: true,
	}

	qListener, err := quic.ListenAddr(r.Proxy.reverseAddress, tlsConf, qConf)
	if err != nil {
		log.Printf("reverse-server: handleUDP listen: %v", err)
		return
	}
	defer qListener.Close()

	fmt.Printf("Listening on %s/%s\n",
		qListener.Addr().Network(), qListener.Addr().String(),
	)

	for {
		qSession, err := qListener.Accept(context.Background())
		if err != nil {
			log.Printf("reverse-server: handleUDP accept: %v", err)
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
			log.Printf("reverse-server: serveQUIC accept: %v", err)
			return
		}
		defer infoStream.Close()
		b, err := ioutil.ReadAll(infoStream)
		if err != nil {
			log.Printf("reverse-server: serveQUIC readall: %v", err)
			return
		}
		fmt.Printf("Session %s/%s message: %q\n",
			s.RemoteAddr().Network(), s.RemoteAddr().String(),
			string(b),
		)
	}()

	r.lpmu.Lock()
	listenAddr := net.JoinHostPort("0.0.0.0", strconv.Itoa(r.localPort))
	r.localPort++
	r.lpmu.Unlock()

	sServer, err := socksServer(listenAddr)
	if err != nil {
		log.Printf("reverse-server: serveQUIC socks: %v", err)
		return
	}

	go func(ctx context.Context) {
		<-ctx.Done()
		sServer.Shutdown()
	}(s.Context())

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
		log.Printf("reverse-server: serveQUIC serve: %v", err)
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

		fmt.Printf("Handling %s/%s - %v -> %s/%s\n",
			source.RemoteAddr().Network(), source.RemoteAddr().String(),
			stream.StreamID(),
			"tcp", r.Address(),
		)

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
	stream, ok := rc.udp[source.String()+d.Address()]
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
		rc.udp[source.String()+d.Address()] = stream
		rc.ul.Unlock()

		fmt.Printf("Handling %s/%s - %v -> %s/%s\n",
			source.Network(), source.String(),
			stream.StreamID(),
			"udp", d.Address(),
		)

		go rc.handleIncoming(source, stream)
	}

	return writeMessage(stream, d.Data)
}

func (rc *reverseConn) handleIncoming(local *net.UDPAddr, s quic.Stream) {
	defer s.Close()

	a, addr, port, err := socks5.ParseAddress(local.String())
	if err != nil {
		log.Printf("reverse-server: handleIncoming parse: %v", err)
		return
	}
	for {
		b, err := readMessage(s)
		if err != nil {
			log.Printf("reverse-server: handleIncoming read: %v", err)
			return
		}
		d := socks5.NewDatagram(a, addr, port, b)
		_, err = rc.ss.UDPConn.WriteToUDP(d.Bytes(), local)
		if err != nil {
			log.Printf("reverse-server: handleIncoming write: %v", err)
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
