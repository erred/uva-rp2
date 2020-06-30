package main

import (
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/pion/stun"
	"github.com/pion/turn/v2"
)

const (
	parallel = 2048
)

var (
	auths = [][]string{
		{"admin", "password"},
		{"user", "pass"},
		{"user", "password"},
		{"ninefingers", "youhavetoberealistic"},
		{"webrtc", "turnpassword"},
	}
)

type Result struct {
	addr    string
	soft    string
	auth    string
	san     string
	rdns    string
	asn     string
	asnname string
}

func newResult(rr []string) Result {
	// auth := ""
	// if rr[2] == "false" {
	// 	auth = "user:pass"
	// }
	// rdns := ""
	// if rr[3] != "unknown" {
	// 	rdns = rr[3]
	// }
	//
	// return Result{
	// 	addr:    rr[0],
	// 	auth:    auth,
	// 	rdns:    rdns,
	// 	asn:     rr[4],
	// 	asnname: rr[6],
	// }
	return Result{
		rr[0],
		"",
		rr[1],
		rr[2],
		rr[3],
		rr[4],
		rr[5],
	}
}

func (r Result) csv() []string {
	return []string{r.addr, r.auth, r.soft, r.san, r.rdns, r.asn, r.asnname}
}

func (r *Result) probe() error {
	// var errs []error
	// var success []string
	// for _, auth := range auths {
	// 	ok, err := scantcp(r.addr+":3478", auth[0], auth[1])
	// 	if err != nil {
	// 		errs = append(errs, err)
	// 	} else if ok {
	// 		success = append(success, auth[0]+":"+auth[1])
	// 	}
	// }
	// r.auth = strings.Join(success, " ")
	//
	// san, err := probetls(r.addr + ":5349")
	// if err != nil {
	// 	errs = append(errs, err)
	// }
	// r.san = strings.Join(san, " ")
	// if len(errs) > 0 {
	// 	return fmt.Errorf("%v", errs)
	// }
	r.soft, _ = software(r.addr + ":3478")
	return nil
}

func main() {
	b, err := ioutil.ReadFile("results2.csv")
	if err != nil {
		panic(err)
	}
	records, err := csv.NewReader(bytes.NewReader(b)).ReadAll()
	if err != nil {
		panic(err)
	}
	results := make([]Result, 0, len(records))
	for _, rr := range records[2:] {
		results = append(results, newResult(rr))
	}

	f, err := os.Create("results2.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	w.Write([]string{"IPv4", "Software", "Auth", "TLS SAN", "rDNS", "ASN", "ASN Name"})

	sem := make(chan struct{}, parallel)
	for i := 0; i < parallel; i++ {
		sem <- struct{}{}
	}

	var wg sync.WaitGroup
	for i := range results {
		<-sem
		wg.Add(1)
		if i%parallel == 0 {
			fmt.Printf("progress % 5d/%d\n", i, len(results))
		}

		go func(i int) {
			defer func() {
				sem <- struct{}{}
				wg.Done()
			}()
			err := results[i].probe()
			if err != nil {
				log.Printf("% 5d: probe %s: %v\n", i, results[i].addr, err)
			}
		}(i)
	}

	for i := range results {
		w.Write(results[i].csv())
	}
}

// func scan(addr, user, pass string) (bool, error) {
// 	conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
// 	if err != nil {
// 		return false, fmt.Errorf("listen: %v", err)
// 	}
// 	defer conn.Close()
// 	client, err := turn.NewClient(&turn.ClientConfig{
// 		STUNServerAddr: addr,
// 		TURNServerAddr: addr,
// 		Conn:           conn,
// 		Username:       user,
// 		Password:       pass,
// 		Realm:          "",
// 	})
// 	if err != nil {
// 		return false, fmt.Errorf("newclient: %v", err)
// 	}
// 	defer client.Close()
//
// 	err = client.Listen()
// 	if err != nil {
// 		return false, fmt.Errorf("listen2: %v", err)
// 	}
//
// 	_, err = client.Allocate()
// 	if err != nil {
// 		if strings.Contains(err.Error(), "error 401: Unauthorized") {
// 			return false, nil
// 		}
// 		return false, fmt.Errorf("allocate: %v", err)
// 	}
// 	return true, nil
// }

func scantcp(addr, user, pass string) (bool, error) {
	conn, err := net.Dial("tcp4", addr)
	if err != nil {
		return false, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	client, err := turn.NewClient(&turn.ClientConfig{
		STUNServerAddr: addr,
		TURNServerAddr: addr,
		Conn:           turn.NewSTUNConn(conn),
		Username:       user,
		Password:       pass,
		Realm:          "",
	})
	if err != nil {
		return false, fmt.Errorf("newclient: %v", err)
	}
	defer client.Close()

	err = client.Listen()
	if err != nil {
		return false, fmt.Errorf("listen2: %v", err)
	}

	_, err = client.Allocate()
	if err != nil {
		if strings.Contains(err.Error(), "error 401: Unauthorized") {
			return false, nil
		}
		return false, fmt.Errorf("allocate: %v", err)
	}
	return true, nil
}

func probetls(addr string) ([]string, error) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", addr, tlsConf)
	if err != nil {
		return nil, fmt.Errorf("dial tls: %w", err)
	}
	defer conn.Close()
	cstate := conn.ConnectionState()
	for _, pc := range cstate.PeerCertificates {
		return pc.DNSNames, nil
	}
	return nil, fmt.Errorf("no peer certificates")
}

func software(addr string) (string, error) {
	conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		return "", fmt.Errorf("listen: %v", err)
	}
	defer conn.Close()
	client, err := turn.NewClient(&turn.ClientConfig{
		STUNServerAddr: addr,
		TURNServerAddr: addr,
		Conn:           conn,
		Username:       "user",
		Password:       "pass",
		Realm:          "",
	})
	if err != nil {
		return "", fmt.Errorf("newclient: %v", err)
	}
	defer client.Close()

	err = client.Listen()
	if err != nil {
		return "", fmt.Errorf("listen2: %v", err)
	}

	msg, _ := stun.Build(
		stun.TransactionID,
		stun.NewType(stun.MethodAllocate, stun.ClassRequest),
		stun.Fingerprint,
	)

	a, _ := net.ResolveUDPAddr("udp", addr)

	trRes, err := client.PerformTransaction(msg, a, false)
	if err != nil {
		return "", err
	}

	res := trRes.Msg

	var soft stun.Software
	err = soft.GetFrom(res)
	if err != nil {
		return "", err
	}
	return soft.String(), nil
}
