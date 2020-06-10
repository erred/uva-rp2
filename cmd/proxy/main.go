package main

import (
	"errors"
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
	flag.StringVar(&p.turnUser, "turnUser", "turnpike", "username for TURN user")
	flag.StringVar(&p.turnPass, "turnPass", "turnpike", "password for TURN user")
	flag.StringVar(&p.turnRealm, "turnRealm", "example.com", "realm for TURN user")

	flag.StringVar(&p.mode, "mode", "forward", "forward / reverse-client / reverse-server connection")

	// forward
	flag.StringVar(&p.socksAddress, "socksAddress", "127.0.0.1:1080", "(forward mode) SOCKS5 host:port")

	// reverse
	flag.StringVar(&p.reverseAddress, "reverseAddress", "145.100.104.122:6789", "(reverse) address of reverse-server")
	flag.StringVar(&p.msg, "msg", "hello world", "message in hello messsage")

	// reverse-client
	flag.BoolVar(&p.tcp, "tcp", false, "reverse tcp connection")
	flag.BoolVar(&p.udp, "udp", false, "reverse udp connection")

	// reverse-server
	flag.IntVar(&p.localPort, "localPort", 1080, "first port to create socks proxies on, increases sequentially")

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
	turnRealm   string

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
	localPort int
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
		Realm:          p.realm,
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
	ua, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("udpConn addr: %w", err)
	}
	uc, err := net.ListenUDP("udp4", ua)
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

var (
	ErrNoHeader = errors.New("no header found")
)

func wrapReverse(b []byte, a *net.UDPAddr) []byte {
	ip4 := a.IP.To4()
	return append([]byte{
		ip4[0], ip4[1], ip4[2], ip4[3],
		uint8(a.Port >> 8), uint8(a.Port % 256),
	}, b...)
}

func unwrapReverse(b []byte) (*net.UDPAddr, []byte, error) {
	if len(b) < 6 {
		return nil, nil, ErrNoHeader
	}
	return &net.UDPAddr{
		IP:   net.IPv4(b[0], b[1], b[2], b[3]),
		Port: int(b[4])<<8 + int(b[5]),
		Zone: "",
	}, b[6:], nil
}
