// Package dopou implements DNS over plain old UDP
package dopou

import (
	"context"
	"net"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dnstransport/dnsoverudp"
	"github.com/bassosimone/netx/internal/dox"
)

// Client is a DNS over plain old UDP client
type Client struct {
	transport *dnsoverudp.Transport
}

// NewClient creates a new DoPOU client.
func NewClient(dialer *dialerapi.Dialer, address string) (*Client, error) {
	return &Client{
		transport: dnsoverudp.NewTransport(dialer.Beginning, dialer.Handler, address),
	}, nil
}

// NewResolver creates a new resolver that uses the specified
// server address to resolve domain names over UDP.
func (clnt *Client) NewResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(c context.Context, n string, a string) (net.Conn, error) {
			return clnt.NewConn()
		},
	}
}

// NewConn returns a new dopou pseudo-conn
func (clnt *Client) NewConn() (net.Conn, error) {
	beginning := clnt.transport.Dialer.Beginning
	handler := clnt.transport.Dialer.Handler
	return dox.NewConn(beginning, handler, func(b []byte) dox.Result {
		return clnt.do(b)
	}), nil
}

// RoundTrip implements the dnsx.RoundTripper interface
func (clnt *Client) RoundTrip(query []byte) (reply []byte, err error) {
	return clnt.transport.RoundTrip(query)
}

func (clnt *Client) do(b []byte) (out dox.Result) {
	out.Data, out.Err = clnt.RoundTrip(b)
	return
}
