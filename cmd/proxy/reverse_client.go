package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"

	quic "github.com/lucas-clemente/quic-go"
)

type ReverseClient struct {
	Proxy
}

func NewReverseClient(p *Proxy) *ReverseClient {
	return &ReverseClient{
		Proxy: *p,
	}
}

func (r *ReverseClient) Run(ctx context.Context) {
	_, uconn, err := r.Proxy.connectUDP()
	if err != nil {
		log.Printf("reverse-client: connectUDP connect: %v", err)
		return
	}

	ua, err := net.ResolveUDPAddr("udp4", r.Proxy.reverseAddress)
	if err != nil {
		log.Printf("reverse-client: connectUDP resolve: %v", err)
		return
	}
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-reverse"},
	}
	quicConf := &quic.Config{
		KeepAlive: true,
	}
	qSession, err := quic.Dial(uconn, ua, ua.String(), tlsConf, quicConf)
	if err != nil {
		log.Printf("reverse-client: connectUDP dial: %v", err)
		return
	}

	go func() {
		infoStream, err := qSession.OpenStream()
		if err != nil {
			log.Printf("reverse-client: infoStream open: %v", err)
			return
		}
		defer infoStream.Close()

		_, err = infoStream.Write([]byte(r.msg))
		if err != nil {
			log.Printf("reverse-client: infoStream write: %v", err)
			return
		}
	}()

	for {
		stream, err := qSession.AcceptStream(context.Background())
		if err != nil {
			log.Printf("reverse-client: connectUDP accept: %v", err)
			return
		}
		p, err := readMessage(stream)
		if err != nil {
			log.Printf("reverse-client: connectUDP read proto: %v", err)
			stream.Close()
			continue
		}
		a, err := readMessage(stream)
		if err != nil {
			log.Printf("reverse-client: connectUDP read dst addr: %v", err)
			stream.Close()
			continue
		}
		switch string(p) {
		case "tcp":
			go serveTCP(string(a), stream)
		case "udp":
			go serveUDP(string(a), stream)
		}
	}
}

func serveTCP(a string, s quic.Stream) {
	defer s.Close()

	c, err := net.Dial("tcp4", a)
	if err != nil {
		log.Printf("reverse-client: serveTCP dial: %v", err)
		return
	}
	defer c.Close()

	fmt.Printf("Handling %v -> %s/%s\n",
		s.StreamID(),
		c.RemoteAddr().Network(), c.RemoteAddr().String(),
	)

	err = copyConn(c, s)
	if err != nil {
		log.Printf("reverse-client: serveTCP copy: %v", err)
	}
}

func serveUDP(a string, s quic.Stream) {
	defer s.Close()

	dstAddr, err := net.ResolveUDPAddr("udp4", a)
	if err != nil {
		log.Printf("reverse-client: serveUDP resolve: %v", err)
		return
	}

	l, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		log.Printf("reverse-client: serveUDP listen: %v", err)
		return
	}
	defer l.Close()

	fmt.Printf("Handling %v -> %s/%s\n",
		s.StreamID(),
		dstAddr.Network(), dstAddr.String(),
	)

	errc := make(chan error)
	go func() {
		buf := make([]byte, 65536)
		for {
			n, _, err := l.ReadFrom(buf)
			if err != nil {
				select {
				case errc <- err:
				default:
				}
				return
			}
			err = writeMessage(s, buf[:n])
			if err != nil {
				select {
				case errc <- err:
				default:
				}
				return
			}
		}
	}()
	go func() {
		for {
			b, err := readMessage(s)
			if err != nil {
				select {
				case errc <- err:
				default:
				}
				return
			}
			_, err = l.WriteTo(b, dstAddr)
			if err != nil {
				// handle error
				select {
				case errc <- err:
				default:
				}
				return
			}
		}
	}()

	err = <-errc
	if err != nil {
		log.Printf("reverse-client: serveUDP copy: %v", err)
	}
}
