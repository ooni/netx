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

// NewResolver creates a new resolver that uses the specified
// server address to resolve domain names over TCP.
func NewResolver(dialer *dialerapi.Dialer, address string) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(c context.Context, n string, a string) (net.Conn, error) {
			return newConn(dialer, address)
		},
	}
}

func newConn(dialer *dialerapi.Dialer, address string) (net.Conn, error) {
	return dox.NewConn(dialer.Beginning, dialer.C, func(b []byte) dox.Result {
		return do(dialer, address, b)
	}), nil
}

type plainResult struct {
	conn net.Conn
	err  error
}

func do(dialer *dialerapi.Dialer, address string, b []byte) (out dox.Result) {
	var conn net.Conn
	ch := make(chan plainResult, 1)
	go func() {
		var r plainResult
		r.conn, _, _, r.err = dialer.DialContextEx(
			context.Background(), "tcp", address, true,
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
