# SOCKS over TURN

UvA / SNE/OS3 / RP2

[![License](https://img.shields.io/github/license/seankhliao/uva-rp2.svg?style=flat-square)](LICENSE)

## ToDo

- [x] read RFCs
- [x] find TURN libraries
- [x] setup coturn
- [x] find http/socks libraries
- [x] questions / scope
- [x] write proposal
- [x] evaluate coturn turnutils
- [x] implement rfc6062
- [x] identify TURN servers
- [x] presentation1
- [ ] report
- [ ] presentation2

https://medium.com/tenable-techblog/using-webrtc-ice-servers-for-port-scanning-in-chrome-ce17b19dd474

## Basics

### Keywords

- ICE: complete framework for NAT traversal
- STUN (RFC 5389): request / response of reflexive transport address (public host:port of client after NAT)
- TURN (RFC 5766): relay data when direct p2p is unavailable
- TURN Allocation: port on server for use by client, 10min refresh
- TURN Permissions: server allowed peer, 5min refresh
- TURN Send/Data: basic send/recv
- TURN Channel: low overhead version of Send/Data, 10min refresh
- TURN TCP: uses separate control / data connections for tcp, same refresh

### Concepts

- client uses STUN to find it's publicly reachable address from a STUN server
- client uses TURN to request an Allocation on a TURN server (with authentication)
- client keeps the allocation using Refresh (default 10min), and deletes when done (Refresh 0s)

## Resources

### RFC

key: **important**, ~~obseleted~~

- [1928](https://tools.ietf.org/html/rfc1928) SOCKS Protocol Version 5
- [3264](https://tools.ietf.org/html/rfc3264) An Offer/Answer Model with the Session Description Protocol (SDP)
- [3489](https://tools.ietf.org/html/rfc3489) ~~STUN - Simple Traversal of User Datagram Protocol (UDP) Through Network Address Translators (NATs)~~
- [4566](https://tools.ietf.org/html/rfc4566) SDP: Session Description Protocol
- [5128](https://tools.ietf.org/html/rfc5128) State of Peer-to-Peer (P2P) Communication across Network Address Translators (NATs)
- [5245](https://tools.ietf.org/html/rfc5245) ~~Interactive Connectivity Establishment (ICE): A Protocol for Network Address Translator (NAT) Traversal for Offer/Answer Protocols~~
- [5389](https://tools.ietf.org/html/rfc5389) **Session Traversal Utilities for NAT (STUN)**
- [5766](https://tools.ietf.org/html/rfc5766) **Traversal Using Relays around NAT (TURN): Relay Extensions to Session Traversal Utilities for NAT (STUN)**
- [5769](https://tools.ietf.org/html/rfc5769) Test Vectors for Session Traversal Utilities for NAT (STUN)
- [5780](https://tools.ietf.org/html/rfc5780) NAT Behavior Discovery Using Session Traversal Utilities for NAT (STUN)
- [5928](https://tools.ietf.org/html/rfc5928) Traversal Using Relays around NAT (TURN) Resolution Mechanism
- [6062](https://tools.ietf.org/html/rfc6062) **Traversal Using Relays around NAT (TURN) Extensions for TCP Allocations**
- [6544](https://tools.ietf.org/html/rfc6544) TCP Candidates with Interactive Connectivity Establishment (ICE)
- [7065](https://tools.ietf.org/html/rfc7065) Traversal Using Relays around NAT (TURN) Uniform Resource Identifiers
- [7350](https://tools.ietf.org/html/rfc7350) Datagram Transport Layer Security (DTLS) as Transport for Session Traversal Utilities for NAT (STUN)
- [7376](https://tools.ietf.org/html/rfc7376) Problems with Session Traversal Utilities for NAT (STUN) Long-Term Authentication for Traversal Using Relays around NAT (TURN)
- [8155](https://tools.ietf.org/html/rfc8155) Traversal Using Relays around NAT (TURN) Server Auto Discovery
- [8445](https://tools.ietf.org/html/rfc8445) Interactive Connectivity Establishment (ICE): A Protocol for Network Address Translator (NAT) Traversal

drafts

- [draft-ietf-mmusic-ice-sip-sdp-39](https://tools.ietf.org/html/draft-ietf-mmusic-ice-sip-sdp-39) Session Description Protocol (SDP) Offer/Answer procedures for Interactive Connectivity Establishment (ICE)

### Libraries

#### STUN / TURN

- [coturn/coturn](https://github.com/coturn/coturn) use turnutils?
- [gortc/stun](https://github.com/gortc/stun)
- [gortc/turn](https://github.com/gortc/turn) [#14](https://github.com/gortc/turn/issues/14) RFC 6062 TURN-TCP not implemented
- [pion/ice](https://github.com/pion/ice)
- [pion/stun](https://github.com/pion/stun)
- [pion/turn](https://github.com/pion/turn) [#118](https://github.com/pion/turn/issues/118) RFC 6062 TURN-TCP not implemented, WIP branch [rfc-6062-client](https://github.com/pion/turn/tree/rfc-6062-client)

#### Proxies

- [armon/go-socks5](https://github.com/armon/go-socks5) SOCKS5
- [cybozu-go/usocksd](https://github.com/cybozu-go/usocksd) SOCKS4/5
- [fangdingjun/socks-go](https://github.com/fangdingjun/socks-go) SOCKS4/5
- [txthinking/socks5](https://github.com/txthinking/socks5) SOCKS5

- [net/http/httputil](https://golang.org/pkg/net/http/httputil) HTTP Reverse Proxy
- [elazarl/goproxy](https://github.com/elazarl/goproxy) HTTP Proxy

### Notes

- Enable Security [slack hack](https://www.rtcsec.com/2020/04/01-slack-webrtc-turn-compromise/)
- draft [TURN over WebSocket](https://tools.ietf.org/html/draft-chenxin-behave-turn-websocket-01)
