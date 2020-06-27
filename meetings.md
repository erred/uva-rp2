# turn servers

- coturn
- prosody

# videoconf things

patch chrome:

- chromium-85.0.4158.4/third_party/webrtc/pc/webrtc_sdp.cc:3015
- `RTC_LOG(LS_INFO) << "SDP got " << message;`
- chromium-85.0.4158.4/third_party/webrtc/p2p/base/turn_port.cc:367
- `<< " -turnUser " << credentials_.username << " -turnPass " << credentials_.password;`

apprtc demo!

## zoom

- TURN creds in websocket ex `wss://zoomff1815631148rwg.cloud.zoom.us/webclient/79891738418?dn2=U2VhbiBMaWFv&zak=eyJ6bV9za20iOiJ6bV9vMm0iLCJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJjbGllbnQiLCJ1aWQiOiJXemI3aEVnZFFNQ1U4RG55QW1SYmlRIiwiZHZjaWQiOiIwNmYyYWQ4MDVmZmU0NWMxYmMwNDUyZmIyNTllZTg4ZSIsImlzcyI6IndlYiIsInN0eSI6MSwid2NkIjoidXMwNCIsImNsdCI6MCwic3RrIjoiZmRtNWlReVRHT2RrRjNNYzJjZ29NYnRIcmx4SDRicFVHaEtGX0Z3d1lmby5FZ0lBQUFGeXhfOE85Z0FBSENBZ2JHRTRjMDV2TVZGTlptOVNNbHBGYlZaYWNsY3lWbVpNWW0xVFRFdEpVakFBRERORFFrRjFiMmxaVXpOelBRUjFjekEwIiwiZXhwIjoxNTkyNDk0MTgxLCJpYXQiOjE1OTI0OTMyODEsImFpZCI6IlZxdmRmUHR4UnlxbElabnZqSFY5MVEiLCJjaWQiOiIifQ.nceWKrEz24Da6omZRCmv1S42wyswTzEuebWwyKtudcU&ts=1592493281012&auth=uvawNv623k-mGBR3CwTraTL51qRDNhhOVUr8QIswQ6s&trackAuth=YJubfyws2-hOjtRo0pEA-IK6I14rclO5eBv4Oq-Qd3Q&mid=nXi86mGaSa2hXRj4zNlVvA%3D%3D&tid=WEB_c6f13b5da2bd32b684c38db3629b070f&browser=Chrome83&ZM-CID=a8274d5c-9fdb-4b05-89b9-ededbc4519d4&lang=en&_ZM_MTG_TRACK_ID=06f2ad805ffe45c1bc0452fb259ee88e&wccv=2.1.1&rwcAuth=MTU5MjQ5MzI4MTg1OC6apd6w7hw5kWg2wukFEeUMhaWc0El5BrH9k5SGhBl-1A&as_type=2&cfs=0`
- from server `{"body":{"encryptKey":"lQ_G64PpyzMWGuVYRcPkV79aR2N5fxgRZ1QvZbyvLBw"},"evt":7938,"seq":3}`
- from client `{"evt":24321,"offer":{"sdp":"v=0\r\no=- 729096162512733402 2 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\na=group:BUNDLE 0\r\na=msid-semantic: WMS\r\nm=application 9 UDP/DTLS/SCTP webrtc-datachannel\r\nc=IN IP4 0.0.0.0\r\na=ice-ufrag:D54499EB-E3F5-26D6-DF88-44345CB3FDA3\r\na=ice-pwd:z1tsgbEy7rPnSByq0PBoI3Wh\r\na=ice-options:trickle\r\na=fingerprint:sha-256 F5:B3:03:84:99:81:33:44:B0:5B:DD:61:D7:08:ED:41:5A:2E:21:22:52:FE:6D:A6:E0:8A:53:28:E4:E2:A3:C7\r\na=setup:actpass\r\na=mid:0\r\na=sctp-port:5000\r\na=max-message-size:262144\r\n","type":1},"seq":3}`
- from server `~{"answer":{"sdp":"v=0\r\no=- 2027211400 2 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\na=group:BUNDLE 0\r\na=msid-semantic: WMS\r\na=ice-lite\r\na=ice-options:trickle\r\nm=application 8801 UDP/DTLS/SCTP webrtc-datachannel\r\nc=IN IP4 0.0.0.0\r\nb=AS:1638400\r\na=ice-ufrag:824C49B4-7737-C4AA-0033-69AEE94DC5B1\r\na=ice-pwd:gGysQFtKbDuXhUPLSKXsDv\r\na=fingerprint:sha-256 13:FC:DD:8A:ED:5F:6F:33:42:7A:33:FF:4D:73:AC:37:7A:00:FF:DD:00:3F:8F:39:8F:63:F5:BF:F3:4A:38:29\r\na=setup:passive\r\na=mid:0\r\na=sctp-port:5000\r\na=max-message-size:1073741823\r\na=candidate:0 1 UDP 2013266431 18.156.31.148 8801 typ host\r\n","type":1},"evt":24322,"seq":57}`
- ??? combine into password
- hmac-sha1 hex: 4d289feda1bb04391eefe6e3567c64421e9cc66d
- unknown attr 0xc057 000003e7
- sdp?
- rfc8445 sec 7.2.2
- (L initiates) user: RFRAG:LFRAG, pass: RPASS
- ice. rfc8445 7.1

## cisco webex

- same as zoom?
- note: use firefox user agent
- m01txmcs853.webex.com:5004 -turnUser ciscoThinClient -turnPass 1234abcd
- uses tcp? also port 80
- can allocate (udp & tcp), can't bind or send data
- webex uses a lot of retransmits?
- xml in base64 in json https://join-test.webex.com/wbxmjs/api/v1/meetings/generateJoinParams?siteurl=join-test

## slack

need paid plan

## ms teams

skype token?

- ./proxy -turnAddress euaz.turn.teams.microsoft.com:3478 -turnUser AgAAJDO7oPMB1k4b9YBY6Qt1ptEsFa1JniKphnf3ZZwAAAAAmb30iKXQWbfPfIkF2HB7RSusPbs= -turnPass H8xfzS6wMc1h6HylT1a8Pg09SWY=
- udp only

## skype

skype token?

## google meet

- probably hides the creds really well
- https://tools.ietf.org/html/draft-uberti-behave-turn-rest-00
- draft-reddy-behave-turn-auth
- grpc =.=
- can't force turn?

## jitsi

- https://jitsi.github.io/handbook/docs/devops-guide/turn ???
- TURN creds in XMPP websocket ex `wss://meet.jit.si/xmpp-websocket?room=helloworld1234567890`
- `<iq from='meet.jit.si' to='50b3fd23-f023-4d27-b67d-1b4924970984@meet.jit.si/hNdmHm-s' id='67aa5b62-9af9-497f-bd0d-aa44a7ccc5b3:sendIQ' xmlns='jabber:client' type='result'><services xmlns='urn:xmpp:extdisco:1'><service type='stun' host='meet-jit-si-turnrelay.jitsi.net' port='443'/><service password='I/L2ETNnVOjChuTKhFrh8fhJGGs=' transport='udp' ttl='86400' type='turn' username='1592564034' host='meet-jit-si-turnrelay.jitsi.net' port='443'/><service password='I/L2ETNnVOjChuTKhFrh8fhJGGs=' transport='tcp' ttl='86400' type='turns' username='1592564034' host='meet-jit-si-turnrelay.jitsi.net' port='443'/></services></iq>`
- example proxy `./proxy -turnAddress meet-jit-si-turnrelay.jitsi.net:443 -turnUser 1592564034 -turnPass I/L2ETNnVOjChuTKhFrh8fhJGGs=`
- udp only, no tcp
