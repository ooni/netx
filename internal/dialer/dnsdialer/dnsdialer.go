// Package dnsdialer contains a dialer with DNS lookups.
package dnsdialer

import (
	"context"
	"errors"
	"net"

	"github.com/ooni/netx/internal/dialer/dialerbase"
	"github.com/ooni/netx/internal/dialid"
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
	ctx = dialid.WithDialID(ctx) // important to create before lookupHost
	dialID := dialid.ContextDialID(ctx)
	var addrs []string
	addrs, err = d.lookupHost(ctx, onlyhost)
	if err != nil {
		return
	}
	var errorslist []error
	for _, addr := range addrs {
		dialer := dialerbase.New(
			root.Beginning, root.Handler, d.dialer, dialID,
		)
		target := net.JoinHostPort(addr, onlyport)
		conn, err = dialer.DialContext(ctx, network, target)
		if err == nil {
			return
		}
		errorslist = append(errorslist, err)
	}
	err = reduceErrors(errorslist)
	return
}

func reduceErrors(errorslist []error) error {
	if len(errorslist) == 0 {
		return nil
	}
	if len(errorslist) == 1 {
		return errorslist[0]
	}
	// TODO(bassosimone): handle this case in a better way
	return errors.New("all connect attempts failed")
}

func (d *Dialer) lookupHost(
	ctx context.Context, hostname string,
) ([]string, error) {
	if net.ParseIP(hostname) != nil {
		return []string{hostname}, nil
	}
	root := model.ContextMeasurementRootOrDefault(ctx)
	lookupHost := root.LookupHost
	if root.LookupHost == nil {
		lookupHost = d.resolver.LookupHost
	}
	addrs, err := lookupHost(ctx, hostname)
	return addrs, err
}
