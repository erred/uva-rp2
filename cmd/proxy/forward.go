package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/txthinking/socks5"
)

type ForwardProxy struct {
	Proxy

	server *socks5.Server

	// udp
	once   sync.Once
	uconn  net.PacketConn
	mu     sync.Mutex
	uconns map[string]UDPConn
}

func NewForwardProxy(p *Proxy) *ForwardProxy {
	return &ForwardProxy{
		Proxy:  *p,
		uconns: make(map[string]UDPConn),
	}
}

func (f *ForwardProxy) Run() error {
	var err error
	f.server, err = socksServer(f.Proxy.socksAddress)
	if err != nil {
		return fmt.Errorf("forward: %w", err)
	}
	fmt.Printf("SOCKS5 listening tcp/%s udp/%s\n",
		f.server.TCPAddr.String(), f.server.UDPAddr.String(),
	)

	err = f.server.ListenAndServe(f)
	if err != nil {
		return fmt.Errorf("forward: %w", err)
	}
	return nil
}

func (f *ForwardProxy) TCPHandle(s *socks5.Server, c *net.TCPConn, r *socks5.Request) error {
	switch r.Cmd {
	case socks5.CmdConnect:
		rc, err := f.DialTCP(r.Address())
		if err != nil {
			return err
		}

		go io.Copy(rc, c)
		io.Copy(c, rc)

		return nil
	case socks5.CmdUDP:
		caddr, err := r.UDP(c, s.ServerAddr)
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

func (f *ForwardProxy) DialTCP(addr string) (net.Conn, error) {
	panic("unimplemented")
}

func (f *ForwardProxy) UDPHandle(s *socks5.Server, addr *net.UDPAddr, d *socks5.Datagram) error {
	f.once.Do(func() {
		var err error
		f.uconn, _, err = f.Proxy.connectUDP()
		if err != nil {
			log.Fatal("handle udp: ", err)
		}

		go f.HandleIncoming()
	})

	ua, err := net.ResolveUDPAddr("udp", d.Address())
	if err != nil {
		return fmt.Errorf("handle udp resolve addr: %w", err)
	}

	c, ok := f.Get(ua.String())
	if ok {
		c.Write(d.Bytes())
		return nil
	}

	nc := UDPConn{
		addr,
		ua,
		f.uconn,
	}
	f.Add(ua.String(), nc)
	nc.Write(d.Bytes())

	fmt.Printf("Handling %s/%s -> %s/%s\n",
		addr.Network(), addr.String(), ua.Network(), ua.String(),
	)
	return nil
}

func (f *ForwardProxy) Get(address string) (UDPConn, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.uconns[address]
	return c, ok
}

func (f *ForwardProxy) Add(address string, c UDPConn) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.uconns[address] = c
}

func (f *ForwardProxy) HandleIncoming() {
	for {
		buf := make([]byte, 65536)
		n, from, err := f.uconn.ReadFrom(buf)
		if err != nil {
			log.Println("handleIncoming read ", err)
		}

		f.mu.Lock()
		c, ok := f.uconns[from.String()]
		f.mu.Unlock()
		if ok {
			a, addr, port, err := socks5.ParseAddress(c.local.String())
			if err != nil {
				log.Println("handleIncoming parse addr: ", err)
			}

			d := socks5.NewDatagram(a, addr, port, buf[:n])
			f.server.UDPConn.WriteToUDP(d.Bytes(), c.local)
		}
	}
}

type UDPConn struct {
	local  *net.UDPAddr
	remote *net.UDPAddr
	relay  net.PacketConn
}

func (c *UDPConn) Read(p []byte) (int, error) {
	return 0, errors.New("unimplemented UDPConn.Read")
}

func (c *UDPConn) RemoteAddr() net.Addr {
	return c.remote
}

func (c *UDPConn) Write(p []byte) (int, error) {
	return c.relay.WriteTo(p, c.remote)
}
