// Package dopot implements DNS over plain old TCP
package dopot

import (
	"errors"
	"net"
	"time"

	"github.com/bassosimone/netx/internal/dot"
	"github.com/bassosimone/netx/internal/dox"
)

// NewConn creates a new net.PacketConn compatible connection that
// will forward DNS queries to the specified DNS server.
func NewConn(address string) (net.Conn, error) {
	return net.Conn(dox.NewConn(func(b []byte) dox.Result {
		return do(address, b)
	})), nil
}

type plainResult struct {
	conn net.Conn
	err  error
}

func do(address string, b []byte) (out dox.Result) {
	var conn net.Conn
	ch := make(chan plainResult, 1)
	go func() {
		var r plainResult
		r.conn, r.err = net.Dial("tcp", address)
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
