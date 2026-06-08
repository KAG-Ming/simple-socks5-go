package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"slices"
	"strconv"
	"time"
)

const (
	socks5Version = 0x05
	methodNoAuth  = 0x00
	cmdConnect    = 0x01
	atypIPv4      = 0x01
	atypDomain    = 0x03
	atypIPv6      = 0x04
	statusSuccess = 0x00
	statusFailure = 0x01
)

const timeoutDuration = 5 * time.Second

func handleConnection(conn net.Conn) {
	defer conn.Close()

	if err := socks5Auth(conn); err != nil {
		log.Printf("Auth failed from %s: %v", conn.RemoteAddr(), err)
		return
	}

	address, port, err := socks5ReadRequest(conn)
	if err != nil {
		log.Printf("Read request failed from %s: %v", conn.RemoteAddr(), err)
		return
	}

	destConn, err := connectDestServer(conn, address, port)
	if err != nil {
		log.Printf("Connect to dest %s:%d failed: %v", address, port, err)
		return
	}
	defer destConn.Close()

	forward(conn, destConn)
}

func socks5Auth(conn net.Conn) error {
	_ = conn.SetReadDeadline(time.Now().Add(timeoutDuration))

	// receive VERSION and NMETHODS
	var buf [2]byte
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return fmt.Errorf("read header failed: %w", err)
	}

	// verify VERSION == 0x05
	if buf[0] != socks5Version {
		return fmt.Errorf("unsupported version: 0x%02x", buf[0])
	}

	numMethods := int(buf[1])
	// choose a method to authenticate
	var methodsBuf [256]byte
	methods := methodsBuf[:numMethods]
	if _, err := io.ReadFull(conn, methods); err != nil {
		return fmt.Errorf("read methods failed: %w", err)
	}

	if !slices.Contains(methods, methodNoAuth) {
		return errors.New("no acceptable auth methods")
	}

	_ = conn.SetReadDeadline(time.Time{})

	// write back to the client
	if _, err := conn.Write([]byte{socks5Version, methodNoAuth}); err != nil {
		return fmt.Errorf("write auth response failed: %w", err)
	}

	return nil
}

func socks5ReadRequest(conn net.Conn) (string, int, error) {
	_ = conn.SetReadDeadline(time.Now().Add(timeoutDuration))

	// read and validate the fixed 4-byte header
	var buf [4]byte
	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return "", 0, fmt.Errorf("read request header failed: %w", err)
	}

	if buf[0] != socks5Version || buf[1] != cmdConnect || buf[2] != 0x00 {
		return "", 0, errors.New("invalid request header flags")
	}

	// parse the target destination address based on ATYP
	var host string
	atyp := buf[3]
	switch atyp {
	case atypIPv4:
		// ipv4 address has a fixed size of 4 bytes
		var ipBuf [4]byte
		if _, err := io.ReadFull(conn, ipBuf[:]); err != nil {
			return "", 0, fmt.Errorf("read IPv4 failed: %w", err)
		}
		host = net.IP(ipBuf[:]).String()

	case atypDomain:
		// for domain names, the first byte indicates the length of the domain string
		var lenBuf [1]byte
		if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
			return "", 0, fmt.Errorf("read domain length failed: %w", err)
		}
		domainLen := int(lenBuf[0])
		domainBuf := make([]byte, domainLen)
		if _, err := io.ReadFull(conn, domainBuf); err != nil {
			return "", 0, fmt.Errorf("read domain name failed: %w", err)
		}
		host = string(domainBuf)

	case atypIPv6:
		// ipv6 address has a fixed size of 16 bytes
		var ipBuf [16]byte
		if _, err := io.ReadFull(conn, ipBuf[:]); err != nil {
			return "", 0, fmt.Errorf("read IPv6 failed: %w", err)
		}
		host = net.IP(ipBuf[:]).String()

	default:
		return "", 0, fmt.Errorf("unsupported address type: 0x%02x", atyp)
	}

	// read and decode the 2-byte port number
	var portBuf [2]byte
	if _, err := io.ReadFull(conn, portBuf[:]); err != nil {
		return "", 0, fmt.Errorf("read port failed: %w", err)
	}

	_ = conn.SetReadDeadline(time.Time{})

	port := int(binary.BigEndian.Uint16(portBuf[:]))
	return host, port, nil
}

func connectDestServer(conn net.Conn, address string, port int) (net.Conn, error) {
    destAddr := net.JoinHostPort(address, strconv.Itoa(port))
    destConn, err := net.DialTimeout("tcp", destAddr, timeoutDuration)
	// handle connection failure by replying to the client
    if err != nil {
        _, _ = conn.Write([]byte{socks5Version, statusFailure, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})
        return nil, err
    }

	// confirm success to the client
    if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
        destConn.Close()
        return nil, fmt.Errorf("write success reply failed: %w", err)
    }
    
    return destConn, nil
}

func forward(conn, destConn net.Conn) {
	go func() {
		_, _ = io.Copy(destConn, conn)
		destConn.Close()  // close the destination connection when the upload ends
	}()

	_, _ = io.Copy(conn, destConn)
}

func main() {
	addr := "127.0.0.1:1080"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()
	log.Printf("SOCKS5 Server launched, listening at %s...", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept connection failed: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}
