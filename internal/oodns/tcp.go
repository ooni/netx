package oodns

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/ooni/netx/dnsx"
)

type tcpTransport struct {
	address     string
	dialContext func(
		ctx context.Context, network, address string) (net.Conn, error)
}

// NewTransportTCP creates a new TCP Transport
func NewTransportTCP(
	address string, dialContext func(
		ctx context.Context, network, address string) (net.Conn, error),
) dnsx.RoundTripper {
	return &tcpTransport{
		address:     address,
		dialContext: dialContext,
	}
}

// RoundTrip sends a request and receives a response.
func (t *tcpTransport) RoundTrip(query []byte) (reply []byte, err error) {
	return t.RoundTripContext(context.Background(), query)
}

// RoundTripContext is like RoundTrip but with context.
func (t *tcpTransport) RoundTripContext(
	ctx context.Context, query []byte,
) (reply []byte, err error) {
	var (
		msg  []byte
		conn net.Conn
	)
	conn, err = t.dialContext(ctx, "tcp", t.address)
	if err != nil {
		return nil, err
	}
	if err = conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, err
	}
	// Write request
	msg = append(msg, byte(len(query)>>8))
	msg = append(msg, byte(len(query)))
	msg = append(msg, query...)
	if _, err = conn.Write(msg); err != nil {
		return nil, err
	}
	// Read response
	header := make([]byte, 2)
	if _, err = io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	length := int(header[0])<<8 | int(header[1])
	reply = make([]byte, length)
	if _, err = io.ReadFull(conn, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
