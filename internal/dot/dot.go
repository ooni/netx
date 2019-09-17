// Package dot implements DNS over TLS
package dot

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"time"

	"github.com/bassosimone/netx/internal/dox"
)

// NewConn creates a new net.PacketConn compatible connection that
// will forward DNS queries to the specified DoT server.
func NewConn(config *tls.Config, domain string) (net.Conn, error) {
	return net.Conn(dox.NewConn(func(b []byte) dox.Result {
		return do(config, domain, b)
	})), nil
}

type tlsResult struct {
	conn *tls.Conn
	err  error
}

func do(config *tls.Config, domain string, b []byte) (out dox.Result) {
	config.ServerName, config.NextProtos = domain, nil
	var conn *tls.Conn
	ch := make(chan tlsResult, 1)
	go func() {
		var r tlsResult
		r.conn, r.err = tls.Dial("tcp", net.JoinHostPort(domain, "853"), config)
		ch <- r
	}()
	select {
	case <-time.After(10 * time.Second):
		out.Err = errors.New("dot: connect deadline expired")
	case r := <-ch:
		conn, out.Err = r.conn, r.err
	}
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
