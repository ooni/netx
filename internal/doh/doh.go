// Package doh implements DNS over HTTPS
package doh

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"
)

type dohresult struct {
	data []byte
	err  error
}

type dohconn struct {
	ch     chan dohresult
	client *http.Client
	mutex  sync.Mutex
	url    string
	rd     time.Time
	wd     time.Time
}

// NewConn creates a new net.PacketConn compatible connection that
// will forward DNS queries to the specified DoH server.
func NewConn(url string) (conn net.Conn, err error) {
	return net.Conn(&dohconn{
		ch:     make(chan dohresult),
		client: http.DefaultClient,
		url:    url,
	}), nil
}

func (c *dohconn) Close() (err error) {
	return
}

func (c *dohconn) LocalAddr() (addr net.Addr) {
	return
}

func (c *dohconn) Read(b []byte) (n int, err error) {
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
		n, err = copy(b, r.data), r.err
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

func (c *dohconn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	err = net.Error(&net.OpError{
		Op:     "ReadFrom",
		Source: c.LocalAddr(),
		Addr:   c.RemoteAddr(),
		Err:    syscall.ENOTCONN,
	})
	return
}

func (c *dohconn) RemoteAddr() (addr net.Addr) {
	return
}

func (c *dohconn) SetDeadline(t time.Time) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.rd = t
	c.wd = t
	return nil
}

func (c *dohconn) SetReadDeadline(t time.Time) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.rd = t
	return nil
}

func (c *dohconn) SetWriteDeadline(t time.Time) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.wd = t
	return nil
}

func (c *dohconn) Write(b []byte) (n int, err error) {
	// An implementation may be tempted to assume that Write on a newly
	// created UDP socket always succeeds. While this is probably not the
	// case for golang, being defensive never hurts too much.
	go c.lookup(b)
	return len(b), nil
}

func (c *dohconn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	err = net.Error(&net.OpError{
		Op:     "WriteTo",
		Source: c.LocalAddr(),
		Addr:   c.RemoteAddr(),
		Err:    syscall.ENOTCONN,
	})
	return
}

func (c *dohconn) lookup(b []byte) {
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

func (c *dohconn) do(b []byte) (out dohresult) {
	var req *http.Request
	req, out.err = http.NewRequest("POST", c.url, bytes.NewReader(b))
	if out.err != nil {
		return
	}
	req.Header.Set("content-type", "application/dns-message")
	var resp *http.Response
	resp, out.err = c.client.Do(req)
	if out.err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		out.err = errors.New("doh: server returned error")
		return
	}
	if resp.Header.Get("content-type") != "application/dns-message" {
		out.err = errors.New("doh: invalid content-type")
		return
	}
	out.data, out.err = ioutil.ReadAll(resp.Body)
	return
}
