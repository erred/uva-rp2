package main

import (
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/txthinking/socks5"
)

type ReverseServer struct {
	Proxy

	wg   sync.WaitGroup
	errc chan error

	server *socks5.Server

	// from->msg
	msgs map[string]string

	// msg->rconn
	ul      sync.Mutex
	clients map[string]*reverseConn

	// udp
	uconn *net.UDPConn
}

func NewReverseServer(p *Proxy) *ReverseServer {
	return &ReverseServer{
		Proxy:   *p,
		errc:    make(chan error),
		msgs:    make(map[string]string),
		clients: make(map[string]*reverseConn),
		// uconns: make(map[string]*UDPConn),
	}
}

func (r *ReverseServer) Run() {
	go errorPrinter("reverse-server", r.errc)

	// r.wg.Add(1)
	// go r.handleTCP()
	r.wg.Add(1)
	go r.handleUDP()

	r.wg.Wait()
	close(r.errc)
}

func (r *ReverseServer) handleTCP() {
	defer r.wg.Done()

	panic("unimplemeted")

}

func (r *ReverseServer) handleUDP() {
	defer r.wg.Done()

	var err error
	r.uconn, err = udpConn(r.Proxy.reverseAddress)
	if err != nil {
		r.errc <- fmt.Errorf("serve: %w", err)
		return
	}

	fmt.Printf("Waiting on %s/%s\n",
		r.uconn.LocalAddr().Network(), r.uconn.LocalAddr().String(),
	)

	buf := make([]byte, 65536)
	for {
		n, from, err := r.uconn.ReadFromUDP(buf)
		if err != nil {
			r.errc <- fmt.Errorf("handleUDP read: %w", err)
			return
		}

		r.ul.Lock()
		msg, ok := r.msgs[from.String()]
		if !ok {
			msg = string(buf[6:n])
			r.msgs[from.String()] = msg
			// doesn't exist
			rc := &reverseConn{
				errc:   r.errc,
				wg:     &r.wg,
				msg:    msg,
				port:   r.Proxy.localPort,
				uconn:  r.uconn,
				remote: from,
			}
			r.clients[msg] = rc

			r.wg.Add(1)
			go rc.serveUDP()

			r.ul.Unlock()
			continue
		}

		rc, ok := r.clients[msg]
		if !ok {
			panic("not found! " + msg)
		}
		r.ul.Unlock()

		if rc.local == nil {
			// no local conn
			continue
		}

		// exists

		a, addr, port, err := socks5.ParseAddress(rc.remote.String())
		if err != nil {
			r.errc <- fmt.Errorf("handleUDP parse: %w", err)
			return
		}

		d := socks5.NewDatagram(a, addr, port, buf[6:n])

		_, err = rc.server.UDPConn.WriteToUDP(d.Bytes(), rc.local)
		if err != nil {
			r.errc <- fmt.Errorf("handleUDP write: %w", err)
		}
	}
}

type reverseConn struct {
	errc chan error
	wg   *sync.WaitGroup
	once sync.Once

	msg    string
	port   int
	server *socks5.Server
	uconn  *net.UDPConn
	remote *net.UDPAddr
	local  *net.UDPAddr
}

func (rc *reverseConn) TCPHandle(s *socks5.Server, c *net.TCPConn, r *socks5.Request) error {
	switch r.Cmd {
	case socks5.CmdConnect:
		panic("unimplemeted")
		// rc, err := rc.dialTCP(r.Address())
		// if err != nil {
		// 	return err
		// }
		//
		// go io.Copy(rc, c)
		// io.Copy(c, rc)

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

func (rc *reverseConn) UDPHandle(s *socks5.Server, addr *net.UDPAddr, d *socks5.Datagram) error {
	if rc.local == nil {
		rc.local = addr
	}

	ua, err := net.ResolveUDPAddr("udp4", d.Address())
	if err != nil {
		return fmt.Errorf("UDPHandle resolve: %w", err)
	}

	b := wrapReverse(d.Data, ua)

	_, err = rc.uconn.WriteToUDP(b, rc.remote)
	if err != nil {
		return fmt.Errorf("UDPHandle write: %w", err)
	}

	return nil
}

func (rc *reverseConn) init() {
	var err error
	rc.server, err = socksServer(net.JoinHostPort("0.0.0.0", strconv.Itoa(rc.port)))
	if err != nil {
		rc.errc <- fmt.Errorf("serveSOCKS: %w", err)
		return
	}
}

func (rc *reverseConn) serveUDP() {
	defer rc.wg.Done()

	rc.once.Do(rc.init)

	fmt.Printf("SOCKS5 listening %s/%s for %s\n",
		rc.server.UDPAddr.Network(), rc.server.UDPAddr.String(), rc.msg,
	)

	err := rc.server.ListenAndServe(rc)
	if err != nil {
		rc.errc <- fmt.Errorf("serveUDP: %w", err)
		return
	}
}
