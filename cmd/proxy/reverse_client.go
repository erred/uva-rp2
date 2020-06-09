package main

import (
	"fmt"
	"net"
	"sync"
)

type ReverseClient struct {
	Proxy

	wg   sync.WaitGroup
	errc chan error

	// tcp
	tconn net.Conn

	// udp
	uconn net.PacketConn
	lconn *net.UDPConn
}

func NewReverseClient(p *Proxy) *ReverseClient {
	return &ReverseClient{
		Proxy: *p,

		errc: make(chan error),
	}
}

func (r *ReverseClient) Run() {
	go errorPrinter("reverse-server", r.errc)

	if r.Proxy.tcp {
		r.wg.Add(1)
		go r.connectTCP()
	}
	if r.Proxy.udp {
		r.wg.Add(1)
		go r.connectUDP()
	}

	r.wg.Wait()
	close(r.errc)
}

func (r *ReverseClient) connectTCP() {
	defer r.wg.Done()

	panic("unimplemented")
}

func (r *ReverseClient) connectUDP() {
	defer r.wg.Done()

	var err error
	r.uconn, _, err = r.Proxy.connectUDP()
	if err != nil {
		r.errc <- fmt.Errorf("connectUDP connect: %w", err)
		return
	}

	r.lconn, err = udpConn("0.0.0.0:0")
	if err != nil {

	}

	r.wg.Add(1)
	go r.handleUDP()
	r.wg.Add(1)
	go r.localToRelay()

	ua, err := net.ResolveUDPAddr("udp4", r.Proxy.reverseAddress)
	if err != nil {
		r.errc <- fmt.Errorf("connectUDP resolve: %w", err)
		return
	}

	zero := make([]byte, len(r.Proxy.msg)+6)
	copy(zero[6:], r.Proxy.msg)

	_, err = r.uconn.WriteTo(zero, ua)
	if err != nil {
		r.errc <- fmt.Errorf("connectUDP hello: %w", err)
		return
	}
}

func (r *ReverseClient) handleUDP() {
	defer r.wg.Done()

	buf := make([]byte, 65536)
	for {
		n, _, err := r.uconn.ReadFrom(buf)
		if err != nil {
			r.errc <- fmt.Errorf("handleUDP read: %w", err)
			return
		}

		dst, buf, err := unwrapReverse(buf[:n])
		if err != nil {
			r.errc <- fmt.Errorf("handleUDP unwrap: %w", err)
			return
		}

		_, err = r.lconn.WriteToUDP(buf, dst)
		if err != nil {
			r.errc <- fmt.Errorf("handleUDP write: %w", err)
			return
		}
	}
}

func (r *ReverseClient) localToRelay() {
	defer r.wg.Done()

	ua, err := net.ResolveUDPAddr("udp4", r.Proxy.reverseAddress)
	if err != nil {
		r.errc <- fmt.Errorf("localToRelay resolve: %w", err)
		return
	}

	buf := make([]byte, 65542)
	for {
		n, _, err := r.lconn.ReadFromUDP(buf[6:])
		if err != nil {
			r.errc <- fmt.Errorf("localToRelay read: %w", err)
			return
		}
		_, err = r.uconn.WriteTo(buf[:n+6], ua)
		if err != nil {
			r.errc <- fmt.Errorf("localToRelay write: %w", err)
			return
		}
	}
}
