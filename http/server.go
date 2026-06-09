package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	line, method, destAddr, err := parseRequestLine(reader)
	if err != nil {
		log.Printf("parse the first line error: %v", err)
		return
	}

	// connect to the destination server
	destConn, err := net.Dial("tcp", destAddr)
	if err != nil {
		log.Printf("connect to the destination server error: %v", err)
		return
	}
	defer destConn.Close()

	if method == "CONNECT" {
		// https
		_, err = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	} else {
		// http
		_, err = destConn.Write([]byte(line))
	}
	if err != nil {
		log.Printf("connect to the destination server error: %v", err)
		return
	}
	
	forward(reader, conn, destConn)
}

func parseRequestLine(reader *bufio.Reader) (line, method, address string, err error) {
	line, err = reader.ReadString('\n')
	if err != nil {
		return "", "", "", fmt.Errorf("read the first line failed: %v", err)
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", "", fmt.Errorf("malformed request line: %s", line)
	}

	method = fields[0]
	if method == "CONNECT" {
		// https "google.com:443"
		return line, method, fields[1], nil
	} else {
		// http "http://google.com/"
		u, err := url.Parse(fields[1])
		if err != nil {
			return "", "", "", fmt.Errorf("parse address error: %v", err)
		}
		address = u.Host
		// if no port in u.Host, add :80
		if !strings.Contains(address, ":") {
			address = address + ":80"
		}
		return line, method, address, nil
	}
}

func forward(reader io.Reader, conn, destConn net.Conn) {
	go func() {
		// ensure that any cached bytes are drained first before io.Copy falls back to reading from the underlying socket
		_, _ = io.Copy(destConn, reader)
		destConn.Close()
	}()
	_, _ = io.Copy(conn, destConn)
}

func main() {
	addr := "127.0.0.1:8080"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()
	log.Printf("HTTP/HTTPS Server launched, listening at %s...", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept connection failed: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}