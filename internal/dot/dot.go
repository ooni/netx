// Package dot implements DNS over TLS
package dot

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"time"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dox"
)

// Client is a DoT client.
type Client struct {
	address string
	dialer  *dialerapi.Dialer
	sni     string
}

// NewClient creates a new client.
func NewClient(dialer *dialerapi.Dialer, address string) (*Client, error) {
	addrs, err := net.LookupHost(address)
	if err != nil {
		return nil, err
	}
	if len(addrs) < 1 {
		return nil, errors.New("dot: net.LookupHost returned an empty slice")
	}
	return &Client{
		address: addrs[0],
		dialer:  dialer,
		sni:     address,
	}, nil
}

// NewResolver creates a new resolver that uses the specified server
// address, and SNI, to resolve domain names over TLS.
func (clnt *Client) NewResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(c context.Context, n string, a string) (net.Conn, error) {
			return clnt.NewConn()
		},
	}
}

// NewConn creates a new DoT pseudo-conn
func (clnt *Client) NewConn() (net.Conn, error) {
	return dox.NewConn(clnt.dialer.Beginning, clnt.dialer.Handler, func(b []byte) dox.Result {
		return clnt.do(b)
	}), nil
}

type tlsResult struct {
	conn *tls.Conn
	err  error
}

// RoundTrip implements the dnsx.RoundTripper interface
func (clnt *Client) RoundTrip(query []byte) (reply []byte, err error) {
	out := clnt.do(query)
	reply = out.Data
	err = out.Err
	return
}

func (clnt *Client) do(b []byte) (out dox.Result) {
	var conn net.Conn
	conn, out.Err = clnt.dialer.DialTLSWithSNI(
		"tcp", net.JoinHostPort(clnt.address, "853"), clnt.sni,
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
	out.Err = conn.SetDeadline(time.Now().Add(10 * time.Second))
	if out.Err != nil {
		return
	}
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
