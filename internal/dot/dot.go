// Package dot implements DNS over TLS
//
// This code is just a tiny wrapper around dnsovertcp.
package dot

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dnstransport/dnsovertcp"
	"github.com/bassosimone/netx/internal/dox"
)

// Client is a DoT client.
type Client struct {
	transport *dnsovertcp.Transport
}

// NewClient creates a new client.
func NewClient(dialer *dialerapi.Dialer, address string) (*Client, error) {
	return &Client{
		transport: dnsovertcp.NewTransport(dialer.Beginning, dialer.Handler, address),
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
	beginning := clnt.transport.Dialer.Beginning
	handler := clnt.transport.Dialer.Handler
	return dox.NewConn(beginning, handler, func(b []byte) dox.Result {
		return clnt.do(b)
	}), nil
}

type tlsResult struct {
	conn *tls.Conn
	err  error
}

// RoundTrip implements the dnsx.RoundTripper interface
func (clnt *Client) RoundTrip(query []byte) (reply []byte, err error) {
	return clnt.transport.RoundTrip(query)
}

func (clnt *Client) do(b []byte) (out dox.Result) {
	out.Data, out.Err = clnt.RoundTrip(b)
	return
}
