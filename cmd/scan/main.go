package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/pion/turn/v2"
)

func main() {
	var start, count, parallel int
	flag.IntVar(&start, "start", 0, "addr to start")
	flag.IntVar(&count, "count", 333_333_333, "addrs to scan")
	flag.IntVar(&parallel, "parallel", 16384, "parallel queries")
	flag.Parse()

	fmt.Printf("start %v count %v \n", start, count)

	sem := make(chan struct{}, parallel)
	for i := 0; i < parallel; i++ {
		sem <- struct{}{}
	}

	f, err := os.Create("turn-results")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var p, s int
	for d := 0; d < 256; d++ {
		for c := 0; c < 256; c++ {
			for b := 0; b < 256; b++ {
				for a := 0; a < 256; a++ {
					if start > 0 {
						start--
						continue
					}
					p++
					if p > count {
						return
					}

					if a == 0 || a == 10 || a == 127 {
						continue
					} else if a == 192 && b == 168 {
						continue
					}

					addr := fmt.Sprintf("%d.%d.%d.%d:3478", a, b, c, d)
					<-sem
					if p%parallel == 0 {
						log.Println("progress ", p, " success ", s)
					}
					go func() {
						defer func() {
							sem <- struct{}{}
						}()
						err := scan(addr)
						if err == nil {
							fmt.Println("ok ", addr)
							f.Write([]byte(addr + ", noauth\n"))
						} else if strings.Contains(err.Error(), "error 401: Unauthorized") {
							s++
							f.Write([]byte(addr + "\n"))
						} else if strings.Contains(err.Error(), "all retransmissions for") {
							// noop
						} else {
							log.Println(err)
						}
					}()
				}
			}
		}
	}
}

func scan(addr string) error {
	conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return fmt.Errorf("listen: %v", err)
	}
	defer conn.Close()
	client, err := turn.NewClient(&turn.ClientConfig{
		STUNServerAddr: addr,
		TURNServerAddr: addr,
		Conn:           conn,
		Username:       "user",
		Password:       "pass",
		Realm:          "realm",
	})
	if err != nil {
		return fmt.Errorf("newclient: %v", err)
	}
	defer client.Close()

	err = client.Listen()
	if err != nil {
		return fmt.Errorf("listen2: %v", err)
	}

	_, err = client.Allocate()
	if err != nil {
		return fmt.Errorf("allocate: %v", err)
	}
	return nil
}
