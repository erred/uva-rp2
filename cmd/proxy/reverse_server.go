package main

import (
	"fmt"
	"net"
	"sync"
)

type ReverseServer struct {
	Proxy

	wg   sync.WaitGroup
	errc chan error

	// tcp
	tcp map[string]net.Conn

	// udp
	ul     sync.Mutex
	uconn  *net.UDPConn
	uconns map[string]*UDPConn
}

func NewReverseServer(p *Proxy) *ReverseServer {
	return &ReverseServer{
		Proxy:  *p,
		errc:   make(chan error),
		uconns: make(map[string]*UDPConn),
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

	for {
		buf := make([]byte, 65536)
		n, from, err := r.uconn.ReadFromUDP(buf)
		if err != nil {
			r.errc <- fmt.Errorf("serve udp read: %w", err)
			return
		}

		r.ul.Lock()
		c := r.uconns[from.String()]
		r.ul.Unlock()
		if c != nil {
			if c.local == nil {
				continue
			}
			_, err = c.conn.WriteToUDP(buf[:n], c.local)
			if err != nil {
				r.errc <- fmt.Errorf("serve udp write: %w", err)
				return
			}
			continue
		}
		nc, err := udpConn("0.0.0.0:0")
		if err != nil {
			r.errc <- fmt.Errorf("serve udp addr: %w", err)
			return
		}

		fmt.Printf("Handle %s/%s -> %s/%s with msg %s\n",
			nc.LocalAddr().Network(), nc.LocalAddr().String(),
			from.Network(), from.String(),
			string(buf[:n]),
		)

		r.ul.Lock()
		r.uconns[from.String()] = &UDPConn{nc, nil}
		r.ul.Unlock()

		r.wg.Add(1)
		go r.localToRelay(nc, from)
	}
}

func (r *ReverseServer) localToRelay(conn *net.UDPConn, addr *net.UDPAddr) {
	defer r.wg.Done()

	buf := make([]byte, 65536)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			r.errc <- fmt.Errorf("localToRelay read: %w", err)
			return
		}
		_, err = r.uconn.WriteToUDP(buf[:n], addr)
		if err != nil {
			r.errc <- fmt.Errorf("localToRelay write: %w", err)
			return
		}
	}
}

type UDPConn struct {
	conn  *net.UDPConn
	local *net.UDPAddr
}
