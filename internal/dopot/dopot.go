// Package dopot implements DNS over plain old TCP.
//
// This code is just a tiny wrapper around dnsovertcp.
package dopot

import (
	"context"
	"net"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dnstransport/dnsovertcp"
	"github.com/bassosimone/netx/internal/dox"
)

// Client is a DNS over plain old TCP client
type Client struct {
	transport *dnsovertcp.Transport
}

// NewClient creates a new DoPOT client.
func NewClient(dialer *dialerapi.Dialer, address string) (*Client, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	transport := dnsovertcp.NewTransport(dialer.Beginning, dialer.Handler, host)
	transport.NoTLS = true
	transport.Port = port
	return &Client{
		transport: transport,
	}, nil
}

// NewResolver creates a new resolver that uses the specified
// server address to resolve domain names over TCP.
func (clnt *Client) NewResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(c context.Context, n string, a string) (net.Conn, error) {
			return clnt.NewConn()
		},
	}
}

// NewConn returns a new dopot pseudo-conn.
func (clnt *Client) NewConn() (net.Conn, error) {
	beginning := clnt.transport.Dialer.Beginning
	handler := clnt.transport.Dialer.Handler
	return dox.NewConn(beginning, handler, func(b []byte) dox.Result {
		return clnt.do(b)
	}), nil
}

type plainResult struct {
	conn net.Conn
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
