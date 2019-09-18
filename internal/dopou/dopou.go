// Package dopou implements DNS over plain old UDP
package dopou

import (
	"context"
	"net"
	"time"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dox"
)

// NewResolver creates a new resolver that uses the specified
// server address to resolve domain names over UDP.
func NewResolver(dialer *dialerapi.Dialer, address string) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(c context.Context, n string, a string) (net.Conn, error) {
			return NewConn(dialer, address)
		},
	}
}

// NewConn returns a new dopou pseudo-conn
func NewConn(dialer *dialerapi.Dialer, address string) (net.Conn, error) {
	return dox.NewConn(dialer.Beginning, dialer.Handler, func(b []byte) dox.Result {
		return do(dialer, address, b)
	}), nil
}

func do(dialer *dialerapi.Dialer, address string, b []byte) (out dox.Result) {
	var conn net.Conn
	conn, _, _, out.Err = dialer.DialContextEx(
		context.Background(), "udp", address, true,
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
