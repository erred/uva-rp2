# proxy

notes: error handling is not good

## build

```sh
go build
```

## general limitations

no clean shutdown / timeout of connections (resource leak)

## run

### forward

proxy runs a local SOCKS server,
forwarding all connections through a TURN relay.
Effectively uses the TURN relay as a forward proxy.

#### limitations

- TCP refreshes not implemented
- opens new TURN session for every TCP connection (allows retargeting same destination without waiting for timeout)
- reuses same TURN session for all UDP connections (multiple udp clients cannot target same destination, replies are all sent to first client)

#### example

```sh
# minimal, uses dev server
./proxy

# all options
./proxy -mode forward \
  -socksAddress 0.0.0.0:1080 \      # address to listen on
  -turnAddress 5.6.7.8:port \       # address of TURN relay
  -turnUser username \
  -turnPass password \
  -turnRealm realm
```

example output:

```sh
# on startup
SOCKS5 listening tcp/127.0.0.1:1080 udp/127.0.0.1:1080

# TCP
# we don't actually know our reflexive address
# mapped proxy-reflexive-addr -> TURN-allocated-addr
# Handling client-source-addr -> client-dest-addr
TURN mapped ???/??? -> tcp/145.100.104.117:64729
Handling tcp/127.0.0.1:52646 -> tcp/104.196.203.254:5678

# UDP
# mapped proxy-reflexive-addr -> TURN-allocated-addr
# Handling client-source-addr -> client-dest-addr
TURN mapped udp/213.124.209.222:45232 -> udp/145.100.104.117:59624
Handling udp/127.0.0.1:42540 -> udp/104.196.203.254:5678
```

### reverse server

Sits at public address, waiting for incoming connections.
Starts a SOCKS proxy per incoming connection.

#### limitations

- only uses UDP (should not actually cause issues, this part runs over public internet)
- possible race condition on localPort?

#### example

```sh
# minimal
./proxy -mode reverse-server

# all options
./proxy -mode reverse-server \
  -reverseAddress 1.2.3.4:6789 \    # address to listen on for incoming connections
  -localPort 1080                   # allocate socks servers sequentially starting here
```

example output

```sh
# on startup
Waiting on udp/145.100.104.122:6789

# on incoming connections
# accepted turn-allocated-address -> local-listening address
# listening on started-socks-addresses for turn-allocated-address
# session turn-allocated-address message: reverse-client-msg
Accepted session udp/145.100.104.117:64548 -> udp/145.100.104.122:6789
Listening on tcp/0.0.0.0:1080 udp/0.0.0.0:1080 for session udp/145.100.104.117:64548
Session udp/145.100.104.117:64548 message: "hello world"

# on incoming SOCKS connection
# client-source-address - stream id -> client-destination-address
# udp
Handling udp/127.0.0.1:60305 - 5 -> udp/104.196.203.254:5678
# tcp
Handling tcp/127.0.0.1:58996 - 1 -> tcp/104.196.203.254:5678
```

### reverse client

Establishes connection to reverse server, serves connections

#### limitations

- only uses UDP (could fix with running TURN in UDP over TCP mode)

#### example

```sh
# minimal
./proxy -mode reverse-client

# all options
./proxy -mode reverse-client \
  -reverseAddress 1.2.3.4:6789  # address of reverse server
  -msg "hello world" \          # message, (to correlate udp and tcp conn)
  -turnAddress 5.6.7.8:port \
  -turnUser username \
  -turnPass password \
  -turnRealm realm
```

example output:

```sh
# on startup
# mapped client-addr-after-nat -> turn-allocated-addr
TURN mapped udp/213.124.209.222:57057 -> udp/145.100.104.117:53197

# on incoming stream
# handling stream-id -> client-destination-address
Handling 1 -> tcp/104.196.203.254:5678
Handling 5 -> udp/104.196.203.254:5678
```

## dev

```txt
// forward
source -> proxy (This program) -> relay (TURN) -> destination

// reverse
source -> server (This program) -> relay (TURN) -> proxy (This program) -> destination
```
