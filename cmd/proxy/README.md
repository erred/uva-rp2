# proxy

notes: error handling is not good

## build

```
go build
```

## run

### forward

have the TURN server forward the connection

```
# minimal, uses dev server
./proxy

# all options
./proxy -mode forward \
  -socksAddress 0.0.0.0:1080 \
  -turnAddress 5.6.7.8:port \
  -turnUser username \
  -turnPass password \
  -turnRealm realm
```

output:

```
# on startup
# tcp doesn't work atm
SOCKS5 listening tcp/127.0.0.1:1080 udp/127.0.0.1:1080

# on incoming socks connection
# mapped forwarder-addr-after-nat -> turn-allocated-address
# handling local-socks-connection -> socks-target
TURN mapped udp/213.124.209.222:47632 -> udp/145.100.104.117:49176
Handling udp/127.0.0.1:34576 -> udp/104.196.203.254:5678
```

### reverse

#### reverse server

sits at public address, waiting for incoming connections

```
# minimal
./proxy -mode reverse-server

# all options
./proxy -mode reverse-server \
  -reverseAddress 1.2.3.4:6789 \    # address to listen on for incoming connections
  -localPort 1080                   # allocate socks servers sequentially starting here
```

output

```
# on startup
Waiting on udp/145.100.104.122:6789

# on incoming client connection
# listening allocated.socks.server for "message from client"
SOCKS5 listening udp/0.0.0.0:1080 for hello world
```

#### reverse client

establishes connection to reverse server, serves connection(s?)

```
# minimal
./proxy -mode reverse-client -udp -reverseAddress 1.2.3.4:6789

# all options
./proxy -mode reverse-client \
  -udp \                        # start a reverse udp connection
  -reverseAddress 1.2.3.4:6789  # address of reverse server
  -msg "hello world" \          # message, (to correlate udp and tcp conn)
  -turnAddress 5.6.7.8:port \
  -turnUser username \
  -turnPass password \
  -turnRealm realm
```

output

```
# on startup
# mapped client-addr-after-nat -> turn-allocated-addr
TURN mapped udp/213.124.209.222:57057 -> udp/145.100.104.117:53197
```
