package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/pion/stun"
	"github.com/pion/turn/v2"
)

func main() {
	var tcp bool
	var user, pass, addr string
	flag.BoolVar(&tcp, "tcp", false, "use tcp")
	flag.StringVar(&addr, "addr", "0.0.0.0:0", "turn server addr")
	flag.StringVar(&user, "user", "user", "user")
	flag.StringVar(&pass, "pass", "pass", "pass")
	flag.Parse()

	var conn net.PacketConn
	var err error
	if tcp {
		tconn, err := net.Dial("tcp4", addr)
		if err != nil {
			panic(err)
		}
		conn = turn.NewSTUNConn(tconn)
	} else {
		conn, err = net.ListenPacket("udp4", "0.0.0.0:0")
		if err != nil {
			panic(err)
		}
		defer conn.Close()
	}
	client, err := turn.NewClient(&turn.ClientConfig{
		STUNServerAddr: addr,
		TURNServerAddr: addr,
		Conn:           conn,
		Username:       user,
		Password:       pass,
		Realm:          "",
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	err = client.Listen()
	if err != nil {
		panic(err)
	}

	msg, _ := stun.Build(
		stun.TransactionID,
		stun.NewType(stun.MethodAllocate, stun.ClassRequest),
		stun.RawAttribute{
			Type:  stun.AttrRequestedTransport,
			Value: []byte{17}, // UDP
		},
		stun.Fingerprint,
	)

	a, _ := net.ResolveUDPAddr("udp", addr)

	trRes, err := client.PerformTransaction(msg, a, false)
	if err != nil {
		panic(err)
	}

	res := trRes.Msg
	for _, a := range res.Attributes {
		fmt.Println(a.Type.String(), ": ", string(a.Value))
	}

	fmt.Println("\nretry with auth\n")

	var nonce stun.Nonce
	var realm stun.Realm
	var integrity stun.MessageIntegrity
	username := stun.NewUsername(user)
	nonce.GetFrom(res)
	realm.GetFrom(res)
	integrity = stun.NewLongTermIntegrity(
		user, realm.String(), pass,
	)

	msg, _ = stun.Build(
		stun.TransactionID,
		stun.NewType(stun.MethodAllocate, stun.ClassRequest),
		stun.RawAttribute{
			Type:  stun.AttrRequestedTransport,
			Value: []byte{6}, // UDP
		},
		username,
		realm,
		nonce,
		integrity,
		stun.Fingerprint,
	)

	trRes, err = client.PerformTransaction(msg, a, false)
	if err != nil {
		panic(err)
	}

	res = trRes.Msg
	for _, a := range res.Attributes {
		fmt.Println(a.Type.String(), ": ", string(a.Value))
	}

	msg, _ = stun.Build(
		stun.TransactionID,
		stun.NewType(stun.MethodRefresh, stun.ClassRequest),
		username,
		realm,
		nonce,
		integrity,
		stun.Fingerprint,
	)

	trRes, err = client.PerformTransaction(msg, a, false)
	if err != nil {
		panic(err)
	}

	res = trRes.Msg
	for _, a := range res.Attributes {
		fmt.Println(a.Type.String(), ": ", string(a.Value))
	}

	fmt.Println(client.SendBindingRequest())
}
