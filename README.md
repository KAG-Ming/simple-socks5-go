# simple-proxy-go

A collection of minimalist network proxy servers in Go for educational purposes.

## Disclaimer
This is a practice project and NOT production-grade software. It lacks robust error handling and may contain bugs. Do not use it in production.

## Available Implementations

### SOCKS5 (`/socks5`)
A minimalist SOCKS5 proxy server. Detailed explanation in my blog: https://blog.onirexus.com/posts/cs/go/simple-socks5-server-go/
```bash
go run main.go
```
The server will start listening at 127.0.0.1:1080.
