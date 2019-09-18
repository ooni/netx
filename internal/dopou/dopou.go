// Package dopou implements DNS over plain old UDP
package dopou

import (
	"context"
	"net"
	"time"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dox"
)

// Client is a DNS over plain old UDP client
type Client struct {
	address string
	dialer  *dialerapi.Dialer
}

// NewClient creates a new DoPOU client.
func NewClient(dialer *dialerapi.Dialer, address string) (*Client, error) {
	return &Client{
		address: address,
		dialer:  dialer,
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
	return dox.NewConn(clnt.dialer.Beginning, clnt.dialer.Handler, func(b []byte) dox.Result {
		return clnt.do(b)
	}), nil
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
	conn, _, _, out.Err = clnt.dialer.DialContextEx(
		context.Background(), "udp", clnt.address, true,
	)
	if out.Err != nil {
		return
	}
	defer conn.Close()
	out.Err = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if out.Err != nil {
		return
	}
	_, out.Err = conn.Write(b)
	if out.Err != nil {
		return
	}
	out.Data = make([]byte, 1<<17)
	var n int
	n, out.Err = conn.Read(out.Data)
	if out.Err == nil {
		out.Data = out.Data[:n]
	}
	return
}
