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
)

// Transport is a DNS over TCP/TLS model.DNSRoundTripper.
//
// As a known bug, this implementation always creates a new connection
// for each incoming query, thus increasing the response delay.
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
func (t *Transport) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	conn, err := t.dial("tcp", t.address)
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
