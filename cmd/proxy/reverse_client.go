package main

import (
	"fmt"
	"log"
	"net"

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
	var err error
	r.uconn, _, err = r.Proxy.connectUDP()
	if err != nil {
		return fmt.Errorf("connect udp: %w", err)
	}

	go r.handleUDP()

	ua, err := net.ResolveUDPAddr("udp4", r.Proxy.reverseAddress)
	if err != nil {
		return fmt.Errorf("connect udp resolve addr: %w", err)
	}
	_, err = r.uconn.WriteTo([]byte(r.Proxy.msg), ua)
	if err != nil {
		return fmt.Errorf("connect udp send hello: %w", err)
	}

	return nil
}

func (r *ReverseClient) handleUDP() {
	var dst *net.UDPConn

	buf := make([]byte, 65536)
	for {
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
		ua, err := net.ResolveUDPAddr("udp", d.Address())
		if err != nil {
			log.Fatal("handle udp resolve addr: ", err)
		}

		if dst == nil {
			dst, err = net.DialUDP("udp", nil, ua)
			if err != nil {
				log.Fatal("handle udp dial: ", err)
			}

			fmt.Printf("Handle %s/%s -> %s/%s\n",
				from.Network(), from.String(), ua.Network(), ua.String(),
			)

			go func() {
				dstAddr, err := net.ResolveUDPAddr("udp", from.String())
				if err != nil {
					log.Fatal("handle udp resolve: ", err)
				}
				a, addr, port, err := socks5.ParseAddress(dstAddr.String())
				if err != nil {
					log.Fatal("handle udp parse addr: ", err)
				}
				buf := make([]byte, 65536)
				for {
					n, _, err := dst.ReadFromUDP(buf)
					if err != nil {
						log.Fatal("handle udp read dst: ", err)
					}

					d := socks5.NewDatagram(a, addr, port, buf[:n])
					_, err = dst.WriteToUDP(d.Bytes(), dstAddr)
					if err != nil {
						log.Fatal("handle udp write: ", err)
					}
				}
			}()
		}

		_, err = dst.WriteToUDP(d.Data, ua)
		if err != nil {
			log.Fatal("handle udp write dst: ", err)
		}

	}
}
