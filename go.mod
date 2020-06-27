module go.seankhliao.com/uva-rp2

go 1.14

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/lucas-clemente/quic-go v0.17.1
	github.com/miekg/dns v1.1.29
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pion/stun v0.3.5
	github.com/pion/turn/v2 v2.0.3
	github.com/txthinking/runnergroup v0.0.0-20200327135940-540a793bb997 // indirect
	github.com/txthinking/socks5 v0.0.0-20200531111549-252709fcb919
	github.com/txthinking/x v0.0.0-20200330144832-5ad2416896a9 // indirect
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)

// use branch with tcp
replace github.com/pion/turn/v2 => github.com/pion/turn/v2 v2.0.4-0.20200612113204-ba7906ed210e

// replace github.com/pion/turn/v2 => ../turn
