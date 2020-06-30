package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/pion/turn/v2"
	"github.com/txthinking/socks5"
)

func main() {
	p := &Proxy{}
	flag.StringVar(&p.turnAddress, "turnAddress", "145.100.104.117:3478", "TURN server host:port")
	flag.StringVar(&p.turnUser, "turnUser", "turnpike", "username for TURN user")
	flag.StringVar(&p.turnPass, "turnPass", "turnpike", "password for TURN user")
	flag.StringVar(&p.turnRealm, "turnRealm", "example.com", "realm for TURN user")

	flag.StringVar(&p.mode, "mode", "forward", "forward / reverse-client / reverse-server connection")
	flag.BoolVar(&p.tcp, "tcp", false, "use tcp for client-relay in udp connections")
	flag.BoolVar(&p.tls, "tls", false, "use tls for client-relay in udp connections")

	// forward
	flag.StringVar(&p.socksAddress, "socksAddress", "127.0.0.1:1080", "(forward mode) SOCKS5 host:port")

	// reverse
	flag.StringVar(&p.reverseAddress, "reverseAddress", "145.100.104.122:6789", "(reverse) address of reverse-server")
	flag.StringVar(&p.msg, "msg", "hello world", "message in hello messsage")

	// reverse-client
	// flag.BoolVar(&p.tcp, "tcp", false, "reverse tcp connection")
	// flag.BoolVar(&p.udp, "udp", false, "reverse udp connection")

	// reverse-server
	flag.IntVar(&p.localPort, "localPort", 1080, "first port to create socks proxies on, increases sequentially")

	flag.Parse()

	switch p.mode {
	case "forward":
		NewForwardProxy(p).Run(context.Background())
	case "reverse-client":
		NewReverseClient(p).Run(context.Background())
	case "reverse-server":
		NewReverseServer(p).Run(context.Background())
	default:
		log.Fatalf("unknown mode %q", p.mode)
	}
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
	tcp  bool
	tls  bool

	// forward
	socksAddress string

	// reverse
	reverseAddress string

	// reverse-client
	// tcp bool
	// udp bool
	msg string

	// reverse-server
	localPort int
}

func (p *Proxy) connectUDP() (*turn.Client, net.PacketConn, error) {
	var turnIn net.PacketConn
	var err error
	if p.tls {
		turnIn0, err := tls.Dial("tcp4", p.turnAddress, &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("connectUDP dial: %w", err)
		}
		turnIn = turn.NewSTUNConn(turnIn0)
	} else if p.tcp {
		turnIn0, err := net.Dial("tcp4", p.turnAddress)
		if err != nil {
			return nil, nil, fmt.Errorf("connectUDP dial: %w", err)
		}
		turnIn = turn.NewSTUNConn(turnIn0)
	} else {
		turnIn, err = net.ListenPacket("udp4", "0.0.0.0:0")
		if err != nil {
			return nil, nil, fmt.Errorf("connectUDP listen: %w", err)
		}
	}

	turnConfig := &turn.ClientConfig{
		STUNServerAddr: p.turnAddress,
		TURNServerAddr: p.turnAddress,
		Conn:           turnIn,
		Username:       p.turnUser,
		Password:       p.turnPass,
		Realm:          p.turnRealm,
	}

	client, err := turn.NewClient(turnConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("connectUDP turn client: %w", err)
	}
	err = client.Listen()
	if err != nil {
		return nil, nil, fmt.Errorf("connectUDP client listen: %w", err)
	}
	relayConn, err := client.Allocate()
	if err != nil {
		return nil, nil, fmt.Errorf("connectUDP client allocate: %w", err)
	}
	mapped, err := client.SendBindingRequest()
	if err != nil {
		return nil, nil, fmt.Errorf("connectUDP client binding: %w", err)
	}

	fmt.Printf("TURN mapped %s/%s -> %s/%s\n",
		mapped.Network(), mapped.String(),
		relayConn.LocalAddr().Network(), relayConn.LocalAddr().String(),
	)
	return client, relayConn, nil
}

func (p *Proxy) connectTCP(dst string) (*turn.Client, net.Conn, error) {
	var err error
	var controlConn net.Conn

	if p.tls {
		controlConn, err = tls.Dial("tcp4", p.turnAddress, &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("connectTCP dial: %w", err)
		}
	} else {
		controlConn, err = net.Dial("tcp4", p.turnAddress)
		if err != nil {
			return nil, nil, fmt.Errorf("connectTCP dial: %w", err)
		}
	}

	turnConfig := &turn.ClientConfig{
		STUNServerAddr:    p.turnAddress,
		TURNServerAddr:    p.turnAddress,
		Conn:              turn.NewSTUNConn(controlConn),
		Username:          p.turnUser,
		Password:          p.turnPass,
		Realm:             p.turnRealm,
		TransportProtocol: turn.ProtoTCP,
	}

	client, err := turn.NewClient(turnConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("connectTCP turn client: %w", err)
	}
	err = client.Listen()
	if err != nil {
		return nil, nil, fmt.Errorf("connectTCP client listen: %w", err)
	}
	relayConn, err := client.Allocate()
	if err != nil {
		return nil, nil, fmt.Errorf("connectTCP client allocate: %w", err)
	}

	ta, err := net.ResolveTCPAddr("tcp4", dst)
	if err != nil {
		return nil, nil, fmt.Errorf("connectTCP resolve: %w", err)
	}
	cid, err := client.Connect(ta)
	if err != nil {
		return nil, nil, fmt.Errorf("connectTCP connect: %w", err)
	}

	dataConn, err := net.Dial("tcp", p.turnAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("connectTCP dial: %w", err)
	}

	err = client.ConnectionBind(dataConn, cid)
	if err != nil {
		return nil, nil, fmt.Errorf("connectTCP bind: %w", err)
	}

	fmt.Printf("TURN mapped %s/%s -> %s/%s\n",
		"tcp", "???",
		// dataConn.LocalAddr().Network(), dataConn.LocalAddr().String(),
		"tcp", relayConn.LocalAddr().String(),
	)

	return client, dataConn, nil
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

func copyConn(conn1, conn2 io.ReadWriter) error {
	errc := make(chan error)
	go func() {
		_, err := io.Copy(conn2, conn1)
		select {
		case errc <- err:
		default:
		}
	}()
	go func() {
		_, err := io.Copy(conn1, conn2)
		select {
		case errc <- err:
		default:
		}
	}()
	return <-errc
}

func readMessage(r io.Reader) ([]byte, error) {
	buf := make([]byte, 4)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	buf = make([]byte, binary.BigEndian.Uint32(buf))
	_, err = io.ReadFull(r, buf)
	return buf, err
}

func writeMessage(w io.Writer, b []byte) error {
	l := uint32(len(b))
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, l)
	_, err := w.Write(buf)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
