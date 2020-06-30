# scan

scan the internet (IPv4) for TURN servers on udp/3478

## notes

- patch `maxRtxCount` in `client.go` in `pion/turn` to `4` to lower time per instance
- runtime: 12h on 10x 4 core 4GB machines (with patched `turn`)
