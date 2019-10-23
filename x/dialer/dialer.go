// Package dialer contains the dialer
package dialer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/resolver"
)

var nextDialID int64

// ConnHash computes the connection ID
func ConnHash(conn net.Conn) string {
	local := conn.LocalAddr()
	remote := conn.RemoteAddr()
	network := local.Network()
	slug := network + local.String() + remote.String()
	sum := sha256.Sum256([]byte(slug))
	return fmt.Sprintf("%x", sum)
}

// Generic is a generic dialer
type Generic interface {
	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Dialer is a dialer
type Dialer struct {
	beginning   time.Time
	dialer      Generic
	handler     model.Handler
	includeData bool
	resolver    resolver.Generic
}

// New returns a new Dialer
func New(
	beginning time.Time,
	handler model.Handler,
	dialer Generic,
	resolver resolver.Generic,
	includeData bool,
) *Dialer {
	return &Dialer{
		beginning:   beginning,
		dialer:      dialer,
		handler:     handler,
		includeData: includeData,
		resolver:    resolver,
	}
}

// Dial establishes a new connection
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// ErrDialFailed is the error returned when dial fails
type ErrDialFailed struct {
	Errors []error
}

// Error returns the error representation as a string
func (e *ErrDialFailed) Error() string {
	return "dialer.go: multiple dials failed"
}

// DialContext establishes a new connection with context
func (d *Dialer) DialContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	var (
		addrs []string
		err   error
	)
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	dialID := atomic.AddInt64(&nextDialID, 1)
	if net.ParseIP(host) == nil {
		// The resolver needs to know far what dial it's
		// about to do the resolving work
		ctx = resolver.WithDialID(ctx, dialID)
		addrs, err = d.resolver.LookupHost(ctx, host)
	} else {
		addrs, err = append(addrs, host), nil
	}
	if err != nil {
		return nil, err
	}
	var errorlist ErrDialFailed
	for _, addr := range addrs {
		target := net.JoinHostPort(addr, port)
		start := time.Now()
		conn, err := d.dialer.DialContext(ctx, network, target)
		stop := time.Now()
		m := model.Measurement{
			Connect: &model.ConnectEvent{
				ConnHash:      "",
				DialID:        dialID,
				Duration:      stop.Sub(start),
				Error:         err,
				Network:       network,
				RemoteAddress: target,
				Time:          stop.Sub(d.beginning),
			},
		}
		if err == nil {
			m.Connect.ConnHash = ConnHash(conn)
			conn = newConnWrapper(
				conn, d.beginning, d.handler, d.includeData, m.Connect.ConnHash,
			)
		}
		d.handler.OnMeasurement(m)
		if err == nil {
			return conn, nil
		}
		errorlist.Errors = append(errorlist.Errors, err)
	}
	return nil, &errorlist
}

type connWrapper struct {
	net.Conn
	beginning   time.Time
	handler     model.Handler
	hash        string
	includeData bool
}

func newConnWrapper(
	conn net.Conn,
	beginning time.Time,
	handler model.Handler,
	includeData bool,
	hash string,
) *connWrapper {
	return &connWrapper{
		Conn:        conn,
		beginning:   beginning,
		handler:     handler,
		hash:        hash,
		includeData: includeData,
	}
}

func (c *connWrapper) Read(b []byte) (n int, err error) {
	start := time.Now()
	n, err = c.Conn.Read(b)
	stop := time.Now()
	m := model.Measurement{
		Read: &model.ReadEvent{
			ConnHash: c.hash,
			Duration: stop.Sub(start),
			Error:    err,
			NumBytes: int64(n),
			Time:     stop.Sub(c.beginning),
		},
	}
	if c.includeData {
		m.Read.Data = b[:n]
	}
	c.handler.OnMeasurement(m)
	return
}

func (c *connWrapper) Write(b []byte) (n int, err error) {
	start := time.Now()
	n, err = c.Conn.Write(b)
	stop := time.Now()
	m := model.Measurement{
		Write: &model.WriteEvent{
			ConnHash: c.hash,
			Duration: stop.Sub(start),
			Error:    err,
			NumBytes: int64(n),
			Time:     stop.Sub(c.beginning),
		},
	}
	if c.includeData {
		m.Write.Data = b[:n]
	}
	c.handler.OnMeasurement(m)
	return
}
