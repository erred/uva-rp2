# Proposal

Proxying SOCKS over TURN

## Intro

SOCKS is a widely supported proxy protocol with a client-server model,
providing a simple interface useful for NAT and firewall traversal.
SOCKS5 \cite{socks} brings support for proxying both TCP and UDP.

Traversal Using Relays around NAT (TURN) \cite{turn} is also a proxy protocol,
extending Session Traversal Utilities for NAT (STUN) \cite{stun}.
While STUN provides utilities for clients to establish peer to peer connections through NAT and firewalls,
this is not always successful, in which case TURN relays can be used to relay data between the clients.
Given its primary usecase in audio/video communications such as in WebRTC \cite{webrtc}, TURN uses UDP for peer connections.
RFC 6062 \cite{turntcp} specifies an extension to TURN to use TCP connections.

Given the nature of proxies,
operators of TURN relays need to be careful in the design of their network
and in the security policies enforced by the proxy itself.
Failure to do so could result in connections being made into internal networks.

For users of other networks, TURN relays run by public entities,
in particular those used by videoconferencing software,
stand in a privileged position as connections to them
are often allowed to pass through both NAT and firewall due to business needs.
Using these TURN relays as generic proxies could punch through firewalls for a wider class of applications.

## Research Question

The goal of this research is to produce a translation layer acting as a SOCKS server and TURN client,
handling protocol translations and the TURN protocol's permissions system,
offering a way to leverage TURN relays as proxies for a wide range of existing tools.

- How can the translation layer be implemented?
- What are the applications and limitations of the translation layer?
- What can be done by firewalls and TURN relays to prevent abuse of proxying?

## Related Work

From early in its design \cite{turn0}
TURN was recognized to stand at a critical juncture between networks.
The latest RFC \cite{turn} expands on the security considerations when running TURN relays.
Additionally, both an authentication and permissions system is built into the protocol,
as well as recommendations in configuration.

The only notable publicised instance of using TURN relays as a proxy
is an April 2020 report by Enable Security
outlining misconfiguration of Slack's TURN relays
and their internal proxying tool \cite{slack}.

## Methodology

coturn is the most popular and widely deployed TURN relay.
It is also the most feature complete, supporting both TCP and UDP connections.
Testing will be run against a locally hosted instance of coturn.

A cursory search did not find any publicly available libraries
in any programming language that implemented TURN-TCP client side code.
github.com/pion/turn has been identified as a suitable candidate
for implementing TCP support if necessary.

Once a translation layer has been complete,
end to end tests can be run to identify issues
and try out ideas for detection and control.

## References

slack

socks

stun

turn0

turn

turntcp

webrtc
