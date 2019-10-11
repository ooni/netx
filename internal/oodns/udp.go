package oodns

import (
	"context"
	"net"
	"time"

	"github.com/ooni/netx/dnsx"
)

type udpTransport struct {
	address     string
	dialContext func(
		ctx context.Context, network string, address string,
	) (net.Conn, error)
}

// NewTransportUDP creates a new UDP transport
func NewTransportUDP(
	address string, dialContext func(
		ctx context.Context, network string, address string,
	) (net.Conn, error),
) dnsx.RoundTripper {
	return &udpTransport{
		dialContext: dialContext,
		address:     address,
	}
}

// RoundTrip sends a request and receives a response.
func (t *udpTransport) RoundTrip(query []byte) (reply []byte, err error) {
	return t.RoundTripContext(context.Background(), query)
}

// RoundTripContext is like RoundTrip but with context.
func (t *udpTransport) RoundTripContext(
	ctx context.Context, query []byte,
) (reply []byte, err error) {
	// TODO(bassosimone): this function does not honour the context and
	// can therefore run for more time than it should run.
	var conn net.Conn
	conn, err = t.dialContext(ctx, "udp", t.address)
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
