// Package dopot implements DNS over plain old TCP
package dopot

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dot"
	"github.com/bassosimone/netx/internal/dox"
)

// Client is a DNS over plain old TCP client
type Client struct {
	address string
	dialer  *dialerapi.Dialer
}

// NewClient creates a new DoPOT client.
func NewClient(dialer *dialerapi.Dialer, address string) (*Client, error) {
	return &Client{
		address: address,
		dialer:  dialer,
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
	return dox.NewConn(clnt.dialer.Beginning, clnt.dialer.Handler, func(b []byte) dox.Result {
		return clnt.do(b)
	}), nil
}

type plainResult struct {
	conn net.Conn
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
	ch := make(chan plainResult, 1)
	go func() {
		var r plainResult
		r.conn, _, _, r.err = clnt.dialer.DialContextEx(
			context.Background(), "tcp", clnt.address, true,
		)
		ch <- r
	}()
	select {
	case <-time.After(10 * time.Second):
		out.Err = errors.New("dopot: connect deadline expired")
	case r := <-ch:
		conn, out.Err = r.conn, r.err
	}
	if out.Err != nil {
		return
	}
	out = dot.OwnConnAndRoundTrip(conn, b)
	return
}
