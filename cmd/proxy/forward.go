package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/pion/turn/v2"
	"github.com/txthinking/socks5"
)

type ForwardProxy struct {
	Proxy

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
	s, err := socksServer(f.Proxy.socksAddress)
	if err != nil {
		return fmt.Errorf("forward: %w", err)
	}

	err = s.ListenAndServe(f)
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
		turnIn, err := net.ListenPacket("udp4", "0.0.0.0:0")
		if err != nil {
			log.Fatalf("connect udp listen: %v", err)
		}
		turnConfig := &turn.ClientConfig{
			STUNServerAddr: f.Proxy.turnAddress,
			TURNServerAddr: f.Proxy.turnAddress,
			Conn:           turnIn,
			Username:       f.Proxy.turnUser,
			Password:       f.Proxy.turnPass,
		}

		client, err := turn.NewClient(turnConfig)
		if err != nil {
			log.Fatalf("connect udp turn client: %v", err)
		}
		err = client.Listen()
		if err != nil {
			log.Fatalf("connect udp client listen: %v", err)
		}
		f.uconn, err = client.Allocate()
		if err != nil {
			log.Fatalf("connect udp client allocate: %v", err)
		}
	})

	c, ok := f.Get(addr.String())
	if ok {
		c.Write(d.Bytes())
		return nil
	}

	nc := UDPConn{
		addr,
		f.uconn,
		make(chan []byte, 10),
	}

	f.Add(addr.String(), nc)

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
			return
		}

		f.mu.Lock()
		c, ok := f.uconns[from.String()]
		f.mu.Unlock()
		if ok {
			select {
			case c.read <- buf[:n]:
			default:
				// drop excess
			}
		}
	}
}

type UDPConn struct {
	net.Addr
	net.PacketConn
	read chan []byte
}

func (c *UDPConn) Read(p []byte) (int, error) {
	b := <-c.read
	copy(p, b)
	return len(b), nil
}

func (c *UDPConn) RemoteAddr() net.Addr {
	return c.Addr
}

func (c *UDPConn) Write(p []byte) (int, error) {
	return c.PacketConn.WriteTo(p, c.Addr)
}
