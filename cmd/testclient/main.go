package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pion/turn/v2"
)

func main() {
	p := Proxy{
		turnAddress: "145.100.104.117:3478",
		turnUser:    "turnpike",
		turnPass:    "turnpike",
	}

	pconn, err := p.connectUDP()
	if err != nil {
		log.Fatal(err)
	}

	ua, err := net.ResolveUDPAddr("udp", "145.100.104.117:5678")
	if err != nil {
		log.Fatal(err)
	}

	for {
		time.Sleep(time.Second)
		_, err = pconn.WriteTo([]byte("hello"), ua)
		if err != nil {
			log.Fatal(err)
		}
		buf := make([]byte, 65536)
		n, from, err := pconn.ReadFrom(buf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(from.String() + " " + string(buf[:n]))
	}

}

type Proxy struct {
	// turn
	protocol    int
	turnAddress string
	turnUser    string
	turnPass    string

	mode string

	// forward
	socksAddress string
	peerAddress  string

	// reverse-client

	// reverse-server
	localAddress string
}

func (p *Proxy) connectUDP() (net.PacketConn, error) {
	// func (p *Proxy) connectUDP() (net.Conn, error) {
	ua, err := net.ResolveUDPAddr("udp4", "0.0.0.0:0")
	if err != nil {
		return nil, fmt.Errorf("connect udp resolve: %w", err)
	}
	turnIn, err := net.ListenUDP("udp4", ua)
	if err != nil {
		return nil, fmt.Errorf("connect udp listen: %w", err)
	}

	turnConfig := &turn.ClientConfig{
		STUNServerAddr: p.turnAddress,
		TURNServerAddr: p.turnAddress,
		Conn:           turnIn,
		Username:       p.turnUser,
		Password:       p.turnPass,
	}

	client, err := turn.NewClient(turnConfig)
	if err != nil {
		return nil, fmt.Errorf("connect udp turn client: %w", err)
	}
	err = client.Listen()
	if err != nil {
		return nil, fmt.Errorf("connect udp client listen: %w", err)
	}
	relayConn, err := client.Allocate()
	if err != nil {
		return nil, fmt.Errorf("connect udp client allocate: %w", err)
	}
	// mappedAddr, err := client.SendBindingRequest()
	// if err != nil {
	// 	return nil, fmt.Errorf("connect udp client bind: %w", err)
	// }
	return relayConn, nil
	// return &packetConn{mappedAddr, relayConn}, nil
}

type packetConn struct {
	net.Addr
	net.PacketConn
}

func (c *packetConn) Read(b []byte) (n int, err error) {
	n, c.Addr, err = c.PacketConn.ReadFrom(b)
	return
}

func (c *packetConn) Write(b []byte) (n int, err error) {
	n, err = c.PacketConn.WriteTo(b, c.Addr)
	return
}
func (c *packetConn) RemoteAddr() net.Addr {
	return c.Addr
}
