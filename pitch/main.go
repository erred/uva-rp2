package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/pion/logging"
	"github.com/pion/turn/v2"
)

func main() {
	host := flag.String("host", "", "TURN Server name.")
	port := flag.Int("port", 3478, "Listening port.")
	user := flag.String("user", "", "A pair of username and password (e.g. \"user@pass\")")
	realm := flag.String("realm", "pion.ly", "Realm (defaults to \"pion.ly\")")
	ping := flag.Bool("ping", false, "Run ping test")
	flag.Parse()

	if len(*host) == 0 {
		log.Fatalf("'host' is required")
	}

	if len(*user) == 0 {
		log.Fatalf("'user' is required")
	}

	// Dial TURN Server
	turnServerAddr := fmt.Sprintf("%s:%d", *host, *port)
	conn, err := net.Dial("tcp", turnServerAddr)
	if err != nil {
		panic(err)
	}

	cred := strings.Split(*user, "@")

	// Start a new TURN Client and wrap our net.Conn in a STUNConn
	// This allows us to simulate datagram based communication over a net.Conn
	cfg := &turn.ClientConfig{
		STUNServerAddr: turnServerAddr,
		TURNServerAddr: turnServerAddr,
		Conn:           turn.NewSTUNConn(conn),
		Username:       cred[0],
		Password:       cred[1],
		Realm:          *realm,
		LoggerFactory:  logging.NewDefaultLoggerFactory(),
	}

	client, err := turn.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Start listening on the conn provided.
	err = client.Listen()
	if err != nil {
		panic(err)
	}

	// Allocate a relay socket on the TURN server. On success, it
	// will return a net.PacketConn which represents the remote
	// socket.
	relayConn, err := client.Allocate()
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := relayConn.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()

	// The relayConn's local address is actually the transport
	// address assigned on the TURN server.
	log.Printf("relayed-address=%s", relayConn.LocalAddr().String())

	// If you provided `-ping`, perform a ping test agaist the
	// relayConn we have just allocated.
	if *ping {
		err = doPingTest(client, relayConn)
		if err != nil {
			panic(err)
		}
	}
}

func doPingTest(client *turn.Client, relayConn net.PacketConn) error {
	addr := &net.UDPAddr{
		IP:   net.ParseIP("5.9.83.84"),
		Port: 5555,
	}

	// Punch a UDP hole for the relayConn by sending a data to the mappedAddr.
	// This will trigger a TURN client to generate a permission request to the
	// TURN server. After this, packets from the IP address will be accepted by
	// the TURN server.
	_, err := relayConn.WriteTo([]byte("HELLO\n"), addr)
	if err != nil {
		return err
	}

	// Start read-loop on relayConn
	go func() {
		buf := make([]byte, 1500)
		for {
			n, _, readerErr := relayConn.ReadFrom(buf)
			if readerErr != nil {
				break
			}

			fmt.Printf("%s", buf[:n])

			// Echo back
			//			if _, readerErr = relayConn.WriteTo(buf[:n], from); readerErr != nil {
			//				break
			//			}
		}
	}()

	time.Sleep(500 * time.Millisecond)

	for {
		time.Sleep(time.Second)
	}

	return nil
}
