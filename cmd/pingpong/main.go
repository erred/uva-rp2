package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/txthinking/socks5"
)

func main() {
	var socks, pong string
	flag.StringVar(&socks, "socks", "127.0.0.1:1080", "SOCKS server address")
	flag.StringVar(&pong, "pong", "104.196.203.254:5678", "pong server address")

	flag.Parse()

	switch flag.Arg(0) {
	case "ping":
		c, err := socks5.NewClient(socks, "", "", 60, 0, 60)
		if err != nil {
			log.Fatal(err)
		}
		conn, err := c.Dial("udp4", pong)
		if err != nil {
			log.Fatal(err)
		}

		buf := make([]byte, 1000)
		go func() {
			for {
				n, err := conn.Read(buf)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("%s -> %s %s\n", conn.RemoteAddr().String(), conn.LocalAddr().String(), buf[:n])
			}
		}()
		for {
			time.Sleep(time.Second)
			_, err = conn.Write([]byte(time.Now().String() + " ping"))
			if err != nil {
				log.Fatal(err)
			}

		}

	case "pong":
		c, err := net.ListenPacket("udp4", ":5678")
		if err != nil {
			log.Fatal(err)
		}

		buf := make([]byte, 1000)
		for {
			n, addr, err := c.ReadFrom(buf)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("%s -> %s %s\n", addr.String(), c.LocalAddr().String(), buf[:n])

			_, err = c.WriteTo([]byte(time.Now().String()+" pong"), addr)
			if err != nil {
				log.Fatal(err)
			}
		}

		// case "socks":
		// 	s, err := socks5.NewClassicServer("145.100.104.117:1080", "145.100.104.117", "", "", 60, 0, 60, 60)
		// 	if err != nil {
		// 		log.Fatal("sock5 server", err)
		// 	}
		// 	err = s.ListenAndServe(nil)
		// 	if err != nil {
		// 		log.Fatal("sock5 server", err)
		// 	}
	}
}
