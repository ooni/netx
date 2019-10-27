// Package dnsdialer contains a dialer with DNS lookups.
package dnsdialer

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/ooni/netx/internal/dialer/dialerbase"
	"github.com/ooni/netx/internal/dialid"
	"github.com/ooni/netx/internal/transactionid"
	"github.com/ooni/netx/model"
)

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	dialer   model.Dialer
	resolver model.DNSResolver
}

// New creates a new Dialer.
func New(resolver model.DNSResolver, dialer model.Dialer) (d *Dialer) {
	return &Dialer{
		dialer:   dialer,
		resolver: resolver,
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
	root := model.ContextMeasurementRootOrDefault(ctx)
	onlyhost, onlyport, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ctx = dialid.WithDialID(ctx)
	dialID := dialid.ContextDialID(ctx)
	root.Handler.OnMeasurement(model.Measurement{
		ResolveStart: &model.ResolveStartEvent{
			DialID:                 dialID,
			DurationSinceBeginning: time.Now().Sub(root.Beginning),
			Hostname:               onlyhost,
			TransactionID:          transactionid.ContextTransactionID(ctx),
		},
	})
	var addrs []string
	addrs, err = d.lookupHost(ctx, onlyhost)
	root.Handler.OnMeasurement(model.Measurement{
		ResolveDone: &model.ResolveDoneEvent{
			Addresses:              addrs,
			DialID:                 dialID,
			DurationSinceBeginning: time.Now().Sub(root.Beginning),
			Error:                  err,
		},
	})
	if err != nil {
		return
	}
	for _, addr := range addrs {
		dialer := dialerbase.New(
			root.Beginning, root.Handler, d.dialer, dialID,
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

func (d *Dialer) lookupHost(ctx context.Context, hostname string) ([]string, error) {
	if net.ParseIP(hostname) != nil {
		return []string{hostname}, nil
	}
	return d.resolver.LookupHost(ctx, hostname)
}
