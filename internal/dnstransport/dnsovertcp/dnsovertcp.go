// Package dnsovertcp implements DNS over TCP. It is possible to
// use both plaintext TCP and TLS.
package dnsovertcp

import (
	"bufio"
	"context"
	"io"
	"net"
	"time"

	"github.com/m-lab/go/rtx"
	"github.com/ooni/netx/model"
)

// Transport is a DNS over TCP/TLS model.DNSRoundTripper.
//
// As a known bug, this implementation always creates a new connection
// for each incoming query, thus increasing the response delay.
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
func (t *Transport) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	conn, err := t.dialer.DialContext(ctx, "tcp", t.address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return t.doWithConn(conn, query)
}

func (t *Transport) doWithConn(conn net.Conn, query []byte) (reply []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			reply = nil // we already got the error just clear the reply
		}
	}()
	err = conn.SetDeadline(time.Now().Add(10 * time.Second))
	rtx.PanicOnError(err, "conn.SetDeadline failed")
	// Write request
	writer := bufio.NewWriter(conn)
	err = writer.WriteByte(byte(len(query) >> 8))
	rtx.PanicOnError(err, "writer.WriteByte failed for first byte")
	err = writer.WriteByte(byte(len(query)))
	rtx.PanicOnError(err, "writer.WriteByte failed for second byte")
	_, err = writer.Write(query)
	rtx.PanicOnError(err, "writer.Write failed for query")
	err = writer.Flush()
	rtx.PanicOnError(err, "writer.Flush failed")
	// Read response
	header := make([]byte, 2)
	_, err = io.ReadFull(conn, header)
	rtx.PanicOnError(err, "io.ReadFull failed")
	length := int(header[0])<<8 | int(header[1])
	reply = make([]byte, length)
	_, err = io.ReadFull(conn, reply)
	rtx.PanicOnError(err, "io.ReadFull failed")
	return reply, nil
}

// TLSDialerAdapter makes a TLSDialer look like a Dialer
type TLSDialerAdapter struct {
	dialer model.TLSDialer
}

// NewTLSDialerAdapter creates a new TLSDialerAdapter
func NewTLSDialerAdapter(dialer model.TLSDialer) *TLSDialerAdapter {
	return &TLSDialerAdapter{dialer: dialer}
}

// Dial dials a new connection
func (d *TLSDialerAdapter) Dial(network, address string) (net.Conn, error) {
	return d.dialer.DialTLS(network, address)
}

// DialContext is like Dial but with context
func (d *TLSDialerAdapter) DialContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	return d.dialer.DialTLSContext(ctx, network, address)
}
