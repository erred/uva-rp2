package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/txthinking/socks5"
)

type ReverseClient struct {
	Proxy

	wg   sync.WaitGroup
	errc chan error

	// tcp
	tconn net.Conn

	// udp
	uconn net.PacketConn
}

func NewReverseClient(p *Proxy) *ReverseClient {
	return &ReverseClient{
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

	r.wg.Add(1)
	go r.handleUDP()

	ua, err := net.ResolveUDPAddr("udp4", r.Proxy.reverseAddress)
	if err != nil {
		r.errc <- fmt.Errorf("connectUDP resolve: %w", err)
		return
	}
	_, err = r.uconn.WriteTo([]byte(r.Proxy.msg), ua)
	if err != nil {
		r.errc <- fmt.Errorf("connectUDP hello: %w", err)
		return
	}
}

func (r *ReverseClient) handleUDP() {
	defer r.wg.Done()

	var dst *net.UDPConn
	buf := make([]byte, 65536)
	for {
		n, from, err := r.uconn.ReadFrom(buf)
		if err != nil {
			r.errc <- fmt.Errorf("handleUDP read: %w", err)
			return
		}

		d, err := socks5.NewDatagramFromBytes(buf[:n])
		if err != nil {
			r.errc <- fmt.Errorf("handleUDP datagram: %w", err)
			return
		}
		ua, err := net.ResolveUDPAddr("udp", d.Address())
		if err != nil {
			r.errc <- fmt.Errorf("handleUDP resolve: %w", err)
			return
		}

		if dst == nil {
			dst, err = net.DialUDP("udp", nil, ua)
			if err != nil {
				r.errc <- fmt.Errorf("handleUDP dial: %w", err)
				return
			}

			fmt.Printf("Handle %s/%s -> %s/%s\n",
				from.Network(), from.String(), ua.Network(), ua.String(),
			)

			r.wg.Add(1)
			go r.localToRelay(dst, from)
		}

		_, err = dst.WriteToUDP(d.Data, ua)
		if err != nil {
			r.errc <- fmt.Errorf("handleUDP write: %w", err)
			return
		}

	}
}

func (r *ReverseClient) localToRelay(conn *net.UDPConn, remote net.Addr) {
	defer r.wg.Done()

	dstAddr, err := net.ResolveUDPAddr("udp", remote.String())
	if err != nil {
		r.errc <- fmt.Errorf("localToRelay resolve: %w", err)
		return
	}
	a, addr, port, err := socks5.ParseAddress(dstAddr.String())
	if err != nil {
		r.errc <- fmt.Errorf("localToRelay parse: %w", err)
		return
	}

	buf := make([]byte, 65536)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			r.errc <- fmt.Errorf("localToRelay read: %w", err)
			return
		}

		d := socks5.NewDatagram(a, addr, port, buf[:n])

		_, err = r.uconn.WriteTo(d.Bytes(), remote)
		if err != nil {
			r.errc <- fmt.Errorf("localToRelay write: %w", err)
			return
		}
	}
}
