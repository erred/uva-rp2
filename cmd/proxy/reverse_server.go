package main

import (
	"fmt"
	"net"
	"sync"
)

type ReverseServer struct {
	Proxy

	errc chan error

	mu      sync.Mutex
	clients map[string]interface{}
}

func NewReverseServer(p *Proxy) *ReverseServer {
	return &ReverseServer{
		Proxy: *p,
		errc:  make(chan error),
	}
}

func (r *ReverseServer) Run() error {
	r.Serve()
	err := <-r.errc
	if err != nil {
		return fmt.Errorf("reverse-server: %w", err)
	}
	return nil
}

func (r *ReverseServer) Serve() {
	go func() {
		// TODO: handle incoming tcp
	}()

	ua, err := net.ResolveUDPAddr("udp", r.Proxy.reverseAddress)
	if err != nil {
		r.errc <- fmt.Errorf("serve udp addr: %w", err)
		return
	}

	uc, err := net.ListenUDP("udp", ua)
	if err != nil {
		r.errc <- fmt.Errorf("serve udp listen: %w", err)
		return
	}
	for {
		buf := make([]byte, 65536)
		n, from, err := uc.ReadFrom(buf)
		if err != nil {
			r.errc <- fmt.Errorf("serve udp read: %w", err)
			return
		}

		r.mu.Lock()
		c, ok := r.clients[from.String()]
		r.mu.Unlock()
		if ok {
			// TODO: send data to c
			_ = c
			_ = n
			continue
		}

		// TODO: start new socks server
	}
}
