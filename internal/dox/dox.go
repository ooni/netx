// Package dox contains code common to doh, dopot, and dot.
package dox

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/bassosimone/netx/internal/connx"
	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/model"
)

// Result is the response to a DNS request.
type Result struct {
	// Data is the data returned.
	Data []byte

	// Err is the error.
	Err error
}

// RoundTripFunc performs a DNS round trip.
type RoundTripFunc func([]byte) Result

// Conn is a doh/dopot/dot connection
type Conn struct {
	ch    chan Result
	f     RoundTripFunc
	id    int64
	mutex sync.Mutex
	rd    time.Time
	wd    time.Time
}

// NewConn creates a new net.PacketConn compatible connection that
// will forward DNS queries to the specified dot/doh server. The
// specified |f| function will perform the real work, using DoT or
// DoH depending on |f|. Nevertheless, as far as the Go client is
// concerned, this code will always receive an entire DNS query
// in Write and will return back a full response in Read.
func NewConn(beginning time.Time, ch chan model.Measurement, f RoundTripFunc) net.Conn {
	connid := dialerapi.NextConnID()
	conn := net.Conn(&connx.DNSMeasuringConn{
		MeasuringConn: connx.MeasuringConn{
			Conn: &Conn{
				ch: make(chan Result),
				f:  f,
				id: connid,
			},
			Beginning: beginning,
			C:         ch,
			ID:        connid,
		},
	})
	safesend(ch, model.Measurement{
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

// Close closes the connection.
func (c *Conn) Close() (err error) {
	return
}

type doxAddr struct {
	id int64
}

func (doxAddr) Network() string {
	return "dns-over-x"
}

func (d doxAddr) String() string {
	return fmt.Sprintf("%d", d.id)
}

// LocalAddr returns the local address.
func (c *Conn) LocalAddr() net.Addr {
	return &doxAddr{c.id}
}

// Read reads the next DNS response.
func (c *Conn) Read(b []byte) (n int, err error) {
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
		n, err = copy(b, r.Data), r.Err
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

// RemoteAddr is a non implemented stub.
func (c *Conn) RemoteAddr() net.Addr {
	return &doxAddr{c.id}
}

// SetDeadline sets the read and the write deadlines.
func (c *Conn) SetDeadline(t time.Time) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.rd = t
	c.wd = t
	return nil
}

// SetReadDeadline sets the read deadline.
func (c *Conn) SetReadDeadline(t time.Time) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.rd = t
	return nil
}

// SetWriteDeadline sets the write deadline.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.wd = t
	return nil
}

// Write writes the next DNS query.
func (c *Conn) Write(b []byte) (n int, err error) {
	// An implementation may be tempted to assume that Write on a newly
	// created UDP socket always succeeds. While this is probably not the
	// case for golang, being defensive never hurts too much.
	go c.lookup(b)
	return len(b), nil
}

func (c *Conn) lookup(b []byte) {
	// If no-one shows up for reading what we have for them for some time
	// then simply give up sending to the channel.
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	select {
	case c.ch <- c.f(b):
		// NOTHING
	case <-timer.C:
		// NOTHING
	}
}

func safesend(ch chan model.Measurement, m model.Measurement) {
	if ch != nil {
		ch <- m
	}
}
