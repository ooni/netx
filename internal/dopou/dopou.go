// Package dopot implements DNS over plain old UDP
package dopou

import (
	"net"
	"time"

	"github.com/bassosimone/netx/internal/dox"
)

// NewConn creates a new net.PacketConn compatible connection that
// will forward DNS queries to the specified DNS server.
func NewConn(address string) (net.Conn, error) {
	return net.Conn(dox.NewConn(func(b []byte) dox.Result {
		return do(address, b)
	})), nil
}

func do(address string, b []byte) (out dox.Result) {
	var conn net.Conn
	conn, out.Err = net.Dial("udp", address)
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
	out.Data = make([]byte, 512)
	var n int
	n, out.Err = conn.Read(out.Data)
	if out.Err == nil {
		out.Data = out.Data[:n]
	}
	return
}
