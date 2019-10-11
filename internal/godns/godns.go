// Package godns wraps src/net/dnsclient.go.
//
// This functionality is only available in platforms where Go is
// using a pure Go DNS implementation (e.g. on Unix).
//
// The core idea behind this package is to modify a net.Resolver
// to request a pure Go resolver and to Dial using a specific
// factory that creates a pseudoconnection that the Go resolver
// see as a net.PacketConnection implementation.
//
// This means that the Go resolver will call Write and pass to
// the pseudoconnection an already formatted DNS query. In a
// similar fashion, when the Go resolver will Read, it will get
// back a full DNS response from the server.
//
// In turn, the pseudoconnection will use the configured DNS
// transport to actually perform the DNS roundtrip.
//
// This implementation is a cheap way to reuse the builtin Go
// DNS client logic and packing/unpacking logic with different
// transports such as DoT and DoH.
//
// Note that we could have modified Dial to return a MeasuringConn
// wrapping a TCP conn to implement both DNS over TLS and DNS
// over TCP. However, the approach of faking a net.PacketConn is
// superior because each Read/Write transfers a complete DNS
// message. So, we don't to deal with DNS's TCP framing.
//
// This package will eventually be replaced by oodns.
package godns

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ooni/netx/dnsx"
	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/model"
)

// NewClient returns a dnsx.Client implementation that is using
// the specified transport to resolve domain names.
func NewClient(
	beginning time.Time, handler model.Handler, transport dnsx.RoundTripper,
) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(c context.Context, n string, a string) (net.Conn, error) {
			return NewPseudoConn(beginning, handler, transport), nil
		},
	}
}

type godnsResult struct {
	reply []byte
	err   error
}

type pseudoConn struct {
	ch    chan godnsResult
	id    int64
	mutex sync.Mutex
	rd    time.Time
	t     dnsx.RoundTripper
	wd    time.Time
}

// NewPseudoConn creates a new pseudo connection attached to the
// specified transport. This allows a DNS client to write a query
// to the conn to send it, and to read to receive the reply.
func NewPseudoConn(
	beginning time.Time, handler model.Handler, transport dnsx.RoundTripper,
) net.Conn {
	connid := dialerapi.NextConnID()
	conn := net.Conn(&connx.DNSMeasuringConn{
		MeasuringConn: connx.MeasuringConn{
			Conn: &pseudoConn{
				ch: make(chan godnsResult),
				id: connid,
				t:  transport,
			},
			Beginning: beginning,
			Handler:   handler,
			ID:        connid,
		},
	})
	handler.OnMeasurement(model.Measurement{
		Connect: &model.ConnectEvent{
			ConnID:        connid,
			Duration:      0,
			Error:         nil,
			LocalAddress:  conn.LocalAddr().String(),
			Network:       conn.LocalAddr().Network(),
			RemoteAddress: conn.RemoteAddr().String(),
			Time:          time.Now().Sub(beginning),
		},
	})
	return conn
}

func (c *pseudoConn) Close() (err error) {
	return
}

type godnsAddr struct {
	id int64
}

func (godnsAddr) Network() string {
	return "godns-pseudo-conn"
}

func (d godnsAddr) String() string {
	return fmt.Sprintf("%d", d.id)
}

func (c *pseudoConn) LocalAddr() net.Addr {
	return &godnsAddr{c.id}
}

func (c *pseudoConn) Read(b []byte) (n int, err error) {
	ctx := context.Background()
	if !c.rd.IsZero() {
		c.mutex.Lock()
		rd := c.rd
		c.mutex.Unlock()
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, rd)
		defer cancel()
	}
	select {
	case r := <-c.ch:
		n, err = copy(b, r.reply), r.err
	case <-ctx.Done():
		n, err = 0, net.Error(&net.OpError{
			Op:     "Read",
			Source: c.LocalAddr(),
			Addr:   c.RemoteAddr(),
			Err:    ctx.Err(),
		})
	}
	return
}

func (c *pseudoConn) RemoteAddr() net.Addr {
	return &godnsAddr{c.id}
}

func (c *pseudoConn) SetDeadline(t time.Time) (err error) {
	err = c.SetReadDeadline(t)
	if err == nil {
		c.SetWriteDeadline(t)
	}
	return
}

func (c *pseudoConn) SetReadDeadline(t time.Time) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.rd = t
	return nil
}

func (c *pseudoConn) SetWriteDeadline(t time.Time) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.wd = t
	return nil
}

func (c *pseudoConn) Write(b []byte) (n int, err error) {
	// An implementation may be tempted to assume that Write on a newly
	// created UDP socket always succeeds. While this is probably not the
	// case for golang, being defensive never hurts too much.
	go c.lookup(b)
	return len(b), nil
}

func (c *pseudoConn) lookup(b []byte) {
	// If no-one shows up for reading what we have for them for some time
	// then simply give up sending to the channel.
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	select {
	case c.ch <- c.do(b):
		// NOTHING
	case <-timer.C:
		// NOTHING
	}
}

func (c *pseudoConn) do(query []byte) (r godnsResult) {
	r.reply, r.err = c.t.RoundTrip(query)
	return r
}
