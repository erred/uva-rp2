package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/pion/turn/v2"
	"github.com/txthinking/socks5"
)

const (
	protoTCP = 6
	protoUDP = 17

	portTURN = 3478
)

func main() {
	p := &Proxy{}
	flag.StringVar(&p.turnAddress, "turnAddress", "145.100.104.117:3478", "TURN server host:port")
	flag.StringVar(&p.turnUser, "turnUser", "turnpike", "username for TURN server")
	flag.StringVar(&p.turnPass, "turnPass", "turnpike", "password for TURN server")

	flag.StringVar(&p.mode, "mode", "forward", "forward / reverse-client / reverse-server connection")

	// forward
	flag.StringVar(&p.socksAddress, "socksAddress", "127.0.0.1:1080", "(forward mode) SOCKS5 host:port")

	// reverse
	flag.StringVar(&p.reverseAddress, "reverseAddress", "145.100.104.117:6789", "(reverse) address of reverse-server")
	flag.StringVar(&p.msg, "msg", "hello world", "message in hello messsage")

	// reverse-client
	flag.BoolVar(&p.tcp, "tcp", false, "reverse tcp connection")
	flag.BoolVar(&p.udp, "udp", false, "reverse udp connection")

	// reverse-server

	flag.Parse()

	switch p.mode {
	case "forward":
		NewForwardProxy(p).Run()
	case "reverse-client":
		NewReverseClient(p).Run()
	case "reverse-server":
		NewReverseServer(p).Run()
	default:
		log.Fatalf("unknown mode %q", p.mode)
	}
}

type runner interface {
	Run()
}

type Proxy struct {
	// turn
	protocol    int
	turnPort    int
	turnAddress string
	turnUser    string
	turnPass    string

	mode string

	// forward
	socksAddress string

	// reverse
	reverseAddress string

	// reverse-client
	tcp bool
	udp bool
	msg string

	// reverse-server
	localPort    int
	localAddress string
}

func (p *Proxy) connectUDP() (net.PacketConn, net.Addr, error) {
	turnIn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return nil, nil, fmt.Errorf("connect udp listen: %w", err)
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
		return nil, nil, fmt.Errorf("connect udp turn client: %w", err)
	}
	err = client.Listen()
	if err != nil {
		return nil, nil, fmt.Errorf("connect udp client listen: %w", err)
	}
	relayConn, err := client.Allocate()
	if err != nil {
		return nil, nil, fmt.Errorf("connect udp client allocate: %w", err)
	}
	mapped, err := client.SendBindingRequest()
	if err != nil {
		return nil, nil, fmt.Errorf("connect udp client binding: %w", err)
	}

	fmt.Printf("TURN mapped %s/%s -> %s/%s\n",
		mapped.Network(), mapped.String(),
		relayConn.LocalAddr().Network(), relayConn.LocalAddr().String(),
	)
	return relayConn, mapped, nil
}

func socksServer(addr string) (*socks5.Server, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("split socks address: %w", err)
	}
	s, err := socks5.NewClassicServer(addr, host, "", "", 60, 0, 60, 60)
	if err != nil {
		return nil, fmt.Errorf("create socks servers: %w", err)
	}
	return s, nil
}

func udpConn(addr string) (*net.UDPConn, error) {
	ua, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("udpConn addr: %w", err)
	}
	uc, err := net.ListenUDP("udp", ua)
	if err != nil {
		return nil, fmt.Errorf("udpConn listen: %w", err)
	}
	return uc, nil
}

func errorPrinter(prefix string, errc chan error) {
	for err := range errc {
		log.Printf("%s: %v", prefix, err)
	}
}
