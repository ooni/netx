// Package dnsoverudp implements DNS over UDP.
package dnsoverudp

import (
	"context"
	"time"

	"github.com/ooni/netx/model"
)

// Transport is a DNS over UDP model.DNSRoundTripper.
type Transport struct {
	dialer  model.Dialer
	address string
}

// NewTransport creates a new Transport
func NewTransport(dialer model.Dialer, address string) *Transport {
	return &Transport{
		dialer:  dialer,
		address: address,
	}
}

// RoundTrip sends a request and receives a response.
func (t *Transport) RoundTrip(ctx context.Context, query []byte) (reply []byte, err error) {
	conn, err := t.dialer.DialContext(ctx, "udp", t.address)
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

// Network returns the transport network (e.g., doh, dot)
func (t *Transport) Network() string {
	return "udp"
}

// Address returns the upstream server address.
func (t *Transport) Address() string {
	return t.address
}
