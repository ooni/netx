// Package doh implements DNS over HTTPS
package doh

import (
	"context"
	"net"
	"time"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dnstransport/dnsoverhttps"
	"github.com/bassosimone/netx/internal/dox"
	"github.com/bassosimone/netx/model"
)

// Client is a DoH client
type Client struct {
	beginning time.Time
	handler   model.Handler
	transport *dnsoverhttps.Transport
}

// NewClient creates a new client.
func NewClient(dialer *dialerapi.Dialer, address string) (*Client, error) {
	return &Client{
		beginning: dialer.Beginning,
		handler:   dialer.Handler,
		transport: dnsoverhttps.NewTransport(
			dialer.Beginning, dialer.Handler, address,
		),
	}, nil
}

// NewResolver creates a new resolver that uses the specified server
// URL, and SNI, to resolve domain names using DoH.
func (clnt *Client) NewResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(c context.Context, n string, a string) (net.Conn, error) {
			return clnt.NewConn()
		},
	}
}

// NewConn creates a new doh pseudo-conn.
func (clnt *Client) NewConn() (net.Conn, error) {
	return dox.NewConn(clnt.beginning, clnt.handler, func(b []byte) dox.Result {
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
