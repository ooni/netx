// Package dot implements DNS over TLS
package dot

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"net"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dox"
)

// NewResolver creates a new resolver that uses the specified server
// address, and SNI, to resolve domain names over TLS.
func NewResolver(dialer *dialerapi.Dialer, address, sni string) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(c context.Context, n string, a string) (net.Conn, error) {
			return newConn(dialer, address, sni)
		},
	}
}

func newConn(dialer *dialerapi.Dialer, address, sni string) (net.Conn, error) {
	return net.Conn(dox.NewConn(func(b []byte) dox.Result {
		return do(dialer, address, sni, b)
	})), nil
}

type tlsResult struct {
	conn *tls.Conn
	err  error
}

func do(dialer *dialerapi.Dialer, address, sni string, b []byte) (out dox.Result) {
	var conn net.Conn
	conn, out.Err = dialer.DialTLSWithSNI(
		"tcp", net.JoinHostPort(address, "853"), sni,
	)
	if out.Err != nil {
		return
	}
	out = OwnConnAndRoundTrip(conn, b)
	return
}

// OwnConnAndRoundTrip owns conn and performs a stream round trip using it. We
// keep this function public because dopot/dopot.go uses it.
func OwnConnAndRoundTrip(conn net.Conn, b []byte) (out dox.Result) {
	defer conn.Close()
	// Write request
	writer := bufio.NewWriter(conn)
	out.Err = writer.WriteByte(byte(len(b) >> 8))
	if out.Err != nil {
		return
	}
	out.Err = writer.WriteByte(byte(len(b)))
	if out.Err != nil {
		return
	}
	_, out.Err = writer.Write(b)
	if out.Err != nil {
		return
	}
	out.Err = writer.Flush()
	if out.Err != nil {
		return
	}
	// Read response
	header := make([]byte, 2)
	_, out.Err = io.ReadFull(conn, header)
	if out.Err != nil {
		return
	}
	length := int(header[0])<<8 | int(header[1])
	out.Data = make([]byte, length)
	_, out.Err = io.ReadFull(conn, out.Data)
	return
}
