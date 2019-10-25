// Package dnsdialer contains a dialer with DNS lookups.
package dnsdialer

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/internal/dialer/dialerbase"
	"github.com/ooni/netx/model"
)

var nextDialID int64

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	beginning time.Time
	dialer    model.Dialer
	handler   model.Handler
	resolver  model.DNSResolver
}

// New creates a new Dialer.
func New(
	beginning time.Time, handler model.Handler,
	resolver model.DNSResolver, dialer model.Dialer,
) (d *Dialer) {
	return &Dialer{
		beginning: beginning,
		dialer:    dialer,
		handler:   handler,
		resolver:  resolver,
	}
}

// Dial creates a TCP or UDP connection. See net.Dial docs.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialContext is like Dial but the context allows to interrupt a
// pending connection attempt at any time.
func (d *Dialer) DialContext(
	ctx context.Context, network, address string,
) (conn net.Conn, err error) {
	onlyhost, onlyport, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	dialID := atomic.AddInt64(&nextDialID, 1)
	if net.ParseIP(onlyhost) != nil {
		dialer := dialerbase.New(
			d.beginning, d.handler, d.dialer, dialID,
		)
		conn, err = dialer.DialContext(ctx, network, address)
		return
	}
	start := time.Now()
	var addrs []string
	addrs, err = d.resolver.LookupHost(ctx, onlyhost)
	stop := time.Now()
	d.handler.OnMeasurement(model.Measurement{
		Resolve: &model.ResolveEvent{
			Addresses: addrs,
			DialID:    dialID,
			Duration:  stop.Sub(start),
			Error:     err,
			Hostname:  onlyhost,
			Time:      stop.Sub(d.beginning),
		},
	})
	if err != nil {
		return
	}
	for _, addr := range addrs {
		dialer := dialerbase.New(
			d.beginning, d.handler, d.dialer, dialID,
		)
		target := net.JoinHostPort(addr, onlyport)
		conn, err = dialer.DialContext(ctx, network, target)
		if err == nil {
			return
		}
	}
	err = &net.OpError{
		Op:  "dial",
		Net: network,
		Err: errors.New("all connect attempts failed"),
	}
	return
}
