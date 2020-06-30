package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/txthinking/socks5"
)

type ForwardProxy struct {
	Proxy

	server *socks5.Server
	cancel context.CancelFunc

	// udp
	uonce  sync.Once
	uconn  net.PacketConn
	ul     sync.Mutex
	uconns map[string]*PacketConn
}

func NewForwardProxy(p *Proxy) *ForwardProxy {
	return &ForwardProxy{
		Proxy:  *p,
		uconns: make(map[string]*PacketConn),
	}
}

// Run starts a SOCKS5 server, forwarding TCP and UDP connections through the TURN relay
func (f *ForwardProxy) Run(ctx context.Context) {
	ctx, f.cancel = context.WithCancel(ctx)

	var err error
	f.server, err = socksServer(f.Proxy.socksAddress)
	if err != nil {
		log.Printf("forward: server create: %v", err)
		return
	}

	go func() {
		<-ctx.Done()
		err := f.server.Shutdown()
		if err != nil {
			log.Printf("forward: server shutdown:")
		}
	}()

	fmt.Printf("SOCKS5 listening tcp/%s udp/%s\n",
		f.server.TCPAddr.String(), f.server.UDPAddr.String(),
	)

	err = f.server.ListenAndServe(f)
	if err != nil {
		log.Printf("forward: server serve: %v", err)
	}
}

// TCPHandle satisfies socks.Handler.
// Initiates TCP connection to relay/destination and copies data between connections
// r.Address: destination address
func (f *ForwardProxy) TCPHandle(s *socks5.Server, source *net.TCPConn, r *socks5.Request) error {
	switch r.Cmd {
	case socks5.CmdConnect:
		// connect to relay
		client, relay, err := f.Proxy.connectTCP(r.Address())
		if err != nil {
			f.cancel()
			return err
		}
		defer client.Close()
		defer relay.Close()

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

		fmt.Printf("Handling %s/%s -> %s/%s\n",
			source.RemoteAddr().Network(), source.RemoteAddr().String(),
			"tcp", r.Address(),
		)

		// copy data between streams
		return copyConn(relay, source)

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

// UDPHandle satisfies socks.Handler
func (f *ForwardProxy) UDPHandle(s *socks5.Server, source *net.UDPAddr, d *socks5.Datagram) error {
	f.uonce.Do(f.initUDP)

	dstAddr, err := net.ResolveUDPAddr("udp4", d.Address())
	if err != nil {
		return fmt.Errorf("UDPHandle resolve: %w", err)
	}

	f.ul.Lock()
	c := f.uconns[dstAddr.String()]
	f.ul.Unlock()

	if c == nil {
		c = &PacketConn{
			source,
			dstAddr,
		}

		f.ul.Lock()
		f.uconns[dstAddr.String()] = c
		f.ul.Unlock()

		fmt.Printf("Handling %s/%s -> %s/%s\n",
			source.Network(), source.String(), dstAddr.Network(), dstAddr.String(),
		)
	}

	_, err = f.uconn.WriteTo(d.Bytes(), c.destination)
	if err != nil {
		return fmt.Errorf("UDPHandle write: %w", err)
	}

	return nil
}

func (f *ForwardProxy) initUDP() {
	var err error
	_, f.uconn, err = f.Proxy.connectUDP()
	if err != nil {
		log.Printf("forward: initUDP: %v", err)
		f.cancel()
		return
	}

	go f.handleIncoming()
}

func (f *ForwardProxy) handleIncoming() {
	buf := make([]byte, 65536)
	for {
		n, from, err := f.uconn.ReadFrom(buf)
		if err != nil {
			log.Printf("forward: handleIncoming read: %v", err)
			return
		}

		f.ul.Lock()
		c := f.uconns[from.String()]
		f.ul.Unlock()
		if c == nil {
			continue
		}

		a, addr, port, err := socks5.ParseAddress(c.source.String())
		if err != nil {
			log.Printf("forward: handleIncoming parse: %v", err)
			return
		}

		d := socks5.NewDatagram(a, addr, port, buf[:n])
		_, err = f.server.UDPConn.WriteToUDP(d.Bytes(), c.source)
		if err != nil {
			log.Printf("forward: handleIncoming write: %v", err)
			return
		}
	}
}

type PacketConn struct {
	source      *net.UDPAddr
	destination *net.UDPAddr
}
