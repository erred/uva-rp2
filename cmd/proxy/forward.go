package main

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/pion/turn/v2"
	"github.com/txthinking/socks5"
)

type ForwardProxy struct {
	Proxy

	wg   sync.WaitGroup
	errc chan error

	server *socks5.Server

	// tcp
	tonce   sync.Once
	tclient *turn.Client

	// udp
	uonce  sync.Once
	uconn  net.PacketConn
	ul     sync.Mutex
	uconns map[string]*PacketConn
}

func NewForwardProxy(p *Proxy) *ForwardProxy {
	return &ForwardProxy{
		Proxy:  *p,
		errc:   make(chan error),
		uconns: make(map[string]*PacketConn),
	}
}

func (f *ForwardProxy) Run() {
	go errorPrinter("forward", f.errc)

	f.wg.Add(1)
	go f.serveSOCKS()

	f.wg.Wait()
	close(f.errc)
}

func (f *ForwardProxy) serveSOCKS() {
	defer f.wg.Done()

	var err error
	f.server, err = socksServer(f.Proxy.socksAddress)
	if err != nil {
		f.errc <- fmt.Errorf("serveSOCKS: %w", err)
		return
	}

	fmt.Printf("SOCKS5 listening tcp/%s udp/%s\n",
		f.server.TCPAddr.String(), f.server.UDPAddr.String(),
	)

	err = f.server.ListenAndServe(f)
	if err != nil {
		f.errc <- fmt.Errorf("serveSOCKS: %w", err)
		return
	}
}

func (f *ForwardProxy) TCPHandle(s *socks5.Server, c *net.TCPConn, r *socks5.Request) error {
	switch r.Cmd {
	case socks5.CmdConnect:
		f.tonce.Do(f.initTCP)

		rc, err := f.dialTCP(r.Address())
		if err != nil {
			return err
		}

		a, addr, port, err := socks5.ParseAddress(rc.LocalAddr().String())
		if err != nil {
			return err
		}

		reply := socks5.NewReply(socks5.RepSuccess, a, addr, port)
		_, err = reply.WriteTo(c)
		if err != nil {
			return err
		}

		go func() {
			_, err = io.Copy(rc, c)
			if err != nil {
				f.errc <- fmt.Errorf("TCPHandle c -> rc: %w", err)
			}
		}()

		_, err = io.Copy(c, rc)
		if err != nil {
			f.errc <- fmt.Errorf("TCPHandle rc -> c: %w", err)
		}

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

func (f *ForwardProxy) dialTCP(addr string) (net.Conn, error) {
	ta, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return nil, fmt.Errorf("dialTCP resolve: %w", err)
	}

	cid, err := f.tclient.Connect(ta)
	if err != nil {
		return nil, fmt.Errorf("dialTCP connect: %w", err)
	}

	dconn, err := net.Dial("tcp", f.Proxy.turnAddress)
	if err != nil {
		return nil, fmt.Errorf("dialTCP dial: %w", err)
	}

	err = f.tclient.ConnectionBind(dconn, cid)
	if err != nil {
		return nil, fmt.Errorf("dialTCP bind: %w", err)
	}

	return dconn, nil
}

func (f *ForwardProxy) UDPHandle(s *socks5.Server, addr *net.UDPAddr, d *socks5.Datagram) error {
	f.uonce.Do(f.initUDP)

	ua, err := net.ResolveUDPAddr("udp4", d.Address())
	if err != nil {
		return fmt.Errorf("UDPHandle resolve: %w", err)
	}

	f.ul.Lock()
	c := f.uconns[ua.String()]
	f.ul.Unlock()

	if c == nil {
		c = &PacketConn{
			addr,
			ua,
			f.uconn,
		}

		f.ul.Lock()
		f.uconns[ua.String()] = c
		f.ul.Unlock()

		fmt.Printf("Handling %s/%s -> %s/%s\n",
			addr.Network(), addr.String(), ua.Network(), ua.String(),
		)
	}

	_, err = c.relay.WriteTo(d.Bytes(), c.remote)
	if err != nil {
		return fmt.Errorf("UDPHandle write: %w", err)
	}

	return nil
}

func (f *ForwardProxy) initTCP() {
	var err error
	f.tclient, _, err = f.Proxy.connectTCP()
	if err != nil {
		f.errc <- fmt.Errorf("initTCP: %w", err)
		return
	}
}

func (f *ForwardProxy) initUDP() {
	var err error
	f.uconn, _, err = f.Proxy.connectUDP()
	if err != nil {
		f.errc <- fmt.Errorf("initUDP: %w", err)
		return
	}

	f.wg.Add(1)
	go f.handleIncoming()
}

func (f *ForwardProxy) handleIncoming() {
	defer f.wg.Done()

	buf := make([]byte, 65536)
	for {
		n, from, err := f.uconn.ReadFrom(buf)
		if err != nil {
			f.errc <- fmt.Errorf("handleIncoming read: %w", err)
			return
		}

		f.ul.Lock()
		c := f.uconns[from.String()]
		f.ul.Unlock()
		if c == nil {
			continue
		}

		a, addr, port, err := socks5.ParseAddress(c.local.String())
		if err != nil {
			f.errc <- fmt.Errorf("handleIncoming parse: %w", err)
			return
		}

		d := socks5.NewDatagram(a, addr, port, buf[:n])
		_, err = f.server.UDPConn.WriteToUDP(d.Bytes(), c.local)
		if err != nil {
			f.errc <- fmt.Errorf("handleIncoming write: %w", err)
			return
		}
	}
}

type PacketConn struct {
	local  *net.UDPAddr
	remote *net.UDPAddr
	relay  net.PacketConn
}
