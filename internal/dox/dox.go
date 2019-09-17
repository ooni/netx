// Package dox contains code common to doh, dopot, and dot.
package dox

import (
	"context"
	"net"
	"sync"
	"syscall"
	"time"
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
func NewConn(f RoundTripFunc) net.Conn {
	return net.Conn(&Conn{
		ch: make(chan Result),
		f:  f,
	})
}

// Close closes the connection.
func (c *Conn) Close() (err error) {
	return
}

type doxAddr struct {
	id string
}

func (doxAddr) Network() string {
	return "dns-over-x"
}

func (d doxAddr) String() string {
	return d.id
}

// LocalAddr returns the local address.
func (c *Conn) LocalAddr() net.Addr {
	return &doxAddr{"local"}
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

// ReadFrom is a non-implemented stub.
func (c *Conn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	err = net.Error(&net.OpError{
		Op:     "ReadFrom",
		Source: c.LocalAddr(),
		Addr:   c.RemoteAddr(),
		Err:    syscall.ENOTCONN,
	})
	return
}

// RemoteAddr is a non implemented stub.
func (c *Conn) RemoteAddr() net.Addr {
	return &doxAddr{"remote"}
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

// WriteTo is a non implemented stub.
func (c *Conn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	err = net.Error(&net.OpError{
		Op:     "WriteTo",
		Source: c.LocalAddr(),
		Addr:   c.RemoteAddr(),
		Err:    syscall.ENOTCONN,
	})
	return
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
