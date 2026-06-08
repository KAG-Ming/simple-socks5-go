# simple-socks5-go
A simple implementation of SOCKS5 server with Golang. Detailed explanation in my blog: https://blog.onirexus.com/posts/cs/go/simple-socks5-server-go/

## Disclaimer

**This is a learning project, NOT a production-grade implementation.** It is intended solely for practicing network programming and understanding RFC 1928. It lacks robust error handling, secure encryption, comprehensive authentication, and may still contain bugs. Do not use it in a production environment.

## Features Covered

- Protocol Version Validation (SOCKS5 only)
- `NO AUTHENTICATION REQUIRED` method negotiation
- `CONNECT` command handling
- Destination address parsing for IPv4, IPv6, and Domain names
- Concurrent full-duplex data forwarding using Goroutines

## Quick Start
```bash
go run server.go
```
The server will start listening at 127.0.0.1:1080.
