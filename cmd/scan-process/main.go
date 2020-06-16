package main

import (
	"bufio"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/miekg/dns"
)

func main() {
	f, err := os.Open("ip2asn-v4-u32.tsv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = 9
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		panic(err)
	}

	var networks []Network
	var starts []int

	for _, record := range records {
		var st, en int
		fmt.Sscanf(record[0], "%d", &st)
		fmt.Sscanf(record[1], "%d", &en)
		starts = append(starts, st)

		networks = append(networks, Network{
			asn:     record[2],
			country: record[3],
			name:    record[4],
		})
	}

	f, err = os.Create("results.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	err = w.Write([]string{"IPv4", "port", "auth", "addr", "ASN", "Country", "Name"})
	if err != nil {
		panic(err)
	}
	defer w.Flush()

	var res []Res

	fis, err := ioutil.ReadDir("results")
	if err != nil {
		panic(err)
	}
	var c int
	for _, fi := range fis {
		fmt.Println(fi.Name())
		f, err := os.Open(path.Join("results", fi.Name()))
		if err != nil {
			panic(err)
		}
		defer f.Close()
		s := bufio.NewScanner(f)
		for s.Scan() {
			ss := strings.Split(s.Text(), ",")
			auth := "true"
			if len(ss) == 2 {
				auth = "false"
			}

			host, port, err := net.SplitHostPort(ss[0])
			if err != nil {
				panic(err)
			}
			ipv4 := net.ParseIP(host).To4()
			u4 := int(binary.BigEndian.Uint32([]byte(ipv4)))

			addr := lookup(host)

			home := sort.SearchInts(starts, u4) - 1
			res = append(res, Res{
				u4,
				[]string{
					ipv4.String(),
					port,
					auth,
					addr,
					networks[home].asn,
					networks[home].country,
					networks[home].name,
				},
			})
			c++
			if c%100 == 0 {
				fmt.Println("progress ", c)
			}
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].a < res[j].a
	})
	for i := range res {
		w.Write(res[i].r)
	}
}

func lookup(host string) string {
	hp := strings.Split(host, ".")
	q := strings.Join([]string{hp[3], hp[2], hp[1], hp[0], "in-addr", "arpa", ""}, ".")

	for {
		msg, err := query(q)
		if err != nil || len(msg.Answer) == 0 {
			return "unknown"
		}
		switch rr := msg.Answer[0].(type) {
		case *dns.PTR:
			return rr.Ptr
		case *dns.CNAME:
			q = rr.Target
			continue
		default:
			fmt.Printf("%T", rr)
			os.Exit(1)
		}
	}
}

func query(q string) (*dns.Msg, error) {
	c := dns.Client{}

	m := &dns.Msg{}
	m.SetQuestion(q, dns.TypePTR)

	var msg *dns.Msg
	var err error
	for i := 0; i < 3; i++ {
		msg, _, err = c.Exchange(m, "8.8.8.8:53")
		if err != nil {
			log.Println(q, err)
			// time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	return msg, err
}

type Res struct {
	a int
	r []string
}

type Network struct {
	asn     string
	country string
	name    string
}
