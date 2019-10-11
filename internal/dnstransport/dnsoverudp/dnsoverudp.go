// Package dnsoverudp implements DNS over UDP.
//
// This package will be eventually replaced by oodns.
package dnsoverudp

import (
	"context"
	"net"
	"time"

	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/model"
)

// Transport is a DNS over UDP dnsx.RoundTripper.
type Transport struct {
	// Dialer is the dialer to use.
	Dialer *dialerapi.Dialer

	// DialContextEx is the function used to dial. NewTransport
	// initializes it to the namesake method of Dialer.
	DialContextEx func(
		ctx context.Context,
		network string,
		address string,
		requireIP bool,
	) (
		conn *connx.MeasuringConn,
		onlyhost string,
		onlyport string,
		err error,
	)

	// Address is the address of the service.
	Address string
}

// NewTransport creates a new Transport
func NewTransport(beginning time.Time, handler model.Handler, address string) *Transport {
	dialer := dialerapi.NewDialer(beginning, handler)
	return &Transport{
		Dialer:        dialer,
		DialContextEx: dialer.DialContextEx,
		Address:       address,
	}
}

// RoundTrip sends a request and receives a response.
func (t *Transport) RoundTrip(query []byte) (reply []byte, err error) {
	return t.RoundTripContext(context.Background(), query)
}

// RoundTripContext is like RoundTrip but with context.
func (t *Transport) RoundTripContext(
	ctx context.Context, query []byte,
) (reply []byte, err error) {
	// TODO(bassosimone): this function does not honour the context.
	var conn net.Conn
	conn, _, _, err = t.DialContextEx(ctx, "udp", t.Address, true)
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
