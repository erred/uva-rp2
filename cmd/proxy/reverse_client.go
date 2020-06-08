package main

import (
	"fmt"
	"net"

	"github.com/pion/turn/v2"
	"github.com/txthinking/socks5"
)

type ReverseClient struct {
	Proxy

	errc chan error

	tconn  net.Conn
	uconn  net.PacketConn
	server *socks5.Server
}

func NewReverseClient(p *Proxy) *ReverseClient {
	return &ReverseClient{
		errc: make(chan error),
	}
}

func (r *ReverseClient) Run() error {
	var err error
	r.server, err = socksServer(r.Proxy.socksAddress)
	if err != nil {
		return fmt.Errorf("create socks server: %w", err)
	}

	if r.Proxy.tcp {
		err = r.ConnectTCP()
		if err != nil {
			return fmt.Errorf("reverse-client: %w", err)
		}
	}
	if r.Proxy.udp {
		err = r.ConnectUDP()
		if err != nil {
			return fmt.Errorf("reverse-client: %w", err)
		}
	}

	return <-r.errc
}

func (r *ReverseClient) ConnectTCP() error {
	panic("unimplemented")
}

func (r *ReverseClient) ConnectUDP() error {
	turnIn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return fmt.Errorf("connect udp listen: %w", err)
	}
	turnConfig := &turn.ClientConfig{
		STUNServerAddr: r.Proxy.turnAddress,
		TURNServerAddr: r.Proxy.turnAddress,
		Conn:           turnIn,
		Username:       r.Proxy.turnUser,
		Password:       r.Proxy.turnPass,
	}

	client, err := turn.NewClient(turnConfig)
	if err != nil {
		return fmt.Errorf("connect udp turn client: %w", err)
	}
	err = client.Listen()
	if err != nil {
		return fmt.Errorf("connect udp client listen: %w", err)
	}
	relayConn, err := client.Allocate()
	if err != nil {
		return fmt.Errorf("connect udp client allocate: %w", err)
	}

	ua, err := net.ResolveUDPAddr("udp4", r.Proxy.reverseAddress)
	if err != nil {
		return fmt.Errorf("connect udp resolve addr: %w", err)
	}
	_, err = relayConn.WriteTo([]byte(r.Proxy.msg), ua)
	if err != nil {
		return fmt.Errorf("connect udp send hello: %w", err)
	}

	r.uconn = relayConn
	go r.handleUDP()

	return nil
}

func (r *ReverseClient) handleUDP() {
	for {
		buf := make([]byte, 65536)
		n, from, err := r.uconn.ReadFrom(buf)
		if err != nil {
			r.errc <- fmt.Errorf("handle udp read: %w", err)
			return
		}

		d, err := socks5.NewDatagramFromBytes(buf[:n])
		if err != nil {
			r.errc <- fmt.Errorf("handle udp datagram: %w", err)
			return
		}
		// if d.Frag != 0x00 {
		// 	r.errc <- fmt.Errorf("Ignore frag", d.Frag)
		// 	return
		// }
		err = r.server.Handle.UDPHandle(r.server, from.(*net.UDPAddr), d)
		if err != nil {
			r.errc <- fmt.Errorf("handle udp handle: %w", err)
			return
		}
	}
}
