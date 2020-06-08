package main

import (
	"flag"
	"fmt"
	"log"
	"net"

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

	var run runner
	switch p.mode {
	case "forward":
		run = NewForwardProxy(p)
	case "reverse-client":
		run = NewReverseClient(p)
	case "reverse-server":
		run = NewReverseServer(p)
	default:
		log.Fatalf("unknown mode %q", p.mode)
	}
	log.Println(run.Run())
}

type runner interface {
	Run() error
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
