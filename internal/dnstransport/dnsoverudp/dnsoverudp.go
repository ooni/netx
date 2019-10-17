// Package dnsoverudp implements DNS over UDP.
package dnsoverudp

import (
	"errors"
	"net"
	"time"
)

// Transport is a DNS over UDP dnsx.RoundTripper.
type Transport struct {
	dial    func(network, address string) (net.Conn, error)
	address string
}

// NewTransport creates a new Transport
func NewTransport(
	dial func(network, address string) (net.Conn, error),
	address string,
) *Transport {
	return &Transport{
		dial:    dial,
		address: address,
	}
}

// RoundTrip sends a request and receives a response.
func (t *Transport) RoundTrip(query []byte) (reply []byte, err error) {
	address, _, err := net.SplitHostPort(t.address)
	if err != nil {
		return
	}
	if net.ParseIP(address) == nil {
		err = errors.New("dnsoverudp: d.address is not IPv4/IPv6")
		return
	}
	var conn net.Conn
	conn, err = t.dial("udp", t.address)
	if err != nil {
		return
	}
	defer conn.Close()
	err = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if err != nil {
		return
	}
	_, err = conn.Write(query)
	if err != nil {
		return
	}
	reply = make([]byte, 1<<17)
	var n int
	n, err = conn.Read(reply)
	if err == nil {
		reply = reply[:n]
	}
	return
}
