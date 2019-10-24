// Package dnsoverudp implements DNS over UDP.
package dnsoverudp

import (
	"context"
	"net"
	"time"
)

// Transport is a DNS over UDP model.DNSRoundTripper.
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
func (t *Transport) RoundTrip(ctx context.Context, query []byte) (reply []byte, err error) {
	conn, err := t.dial("udp", t.address)
	if err != nil {
		return
	}
	defer conn.Close()
	// Use five seconds timeout like Bionic does. See
	// https://labs.ripe.net/Members/baptiste_jonglez_1/persistent-dns-connections-for-reliability-and-performance
	err = conn.SetDeadline(time.Now().Add(5 * time.Second))
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
