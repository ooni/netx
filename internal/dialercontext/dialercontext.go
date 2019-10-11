// Package dialercontext contains a DNS-aware dialer that only
// implements the DialContext interface.
package dialercontext

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/dialerbase"
	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
)

var nextConnID int64

// NextConnID returns the next connection ID.
func NextConnID() int64 {
	return atomic.AddInt64(&nextConnID, 1)
}

// LookupHostFunc is the type of the function used to lookup
// the addresses of a specific host.
type LookupHostFunc func(context.Context, string) ([]string, error)

// DialHostPortFunc is the type of the function that is actually
// used to dial a connection to a specific host and port.
type DialHostPortFunc func(
	ctx context.Context, handler model.Handler,
	network, onlyhost, onlyport string, connid int64,
) (*connx.MeasuringConn, error)

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	DialHostPort DialHostPortFunc
	LookupHost   LookupHostFunc
	dialer       *dialerbase.Dialer
}

// NewDialer creates a new Dialer.
func NewDialer(beginning time.Time) (d *Dialer) {
	d = &Dialer{
		dialer: dialerbase.NewDialer(
			beginning,
		),
	}
	// This is equivalent to ConfigureDNS("system", "...")
	r := &net.Resolver{
		PreferGo: false,
	}
	d.LookupHost = r.LookupHost
	d.DialHostPort = d.dialer.DialHostPort
	return
}

// DialContext is like net.Dial but the context allows to interrupt a
// pending connection attempt at any time.
func (d *Dialer) DialContext(
	ctx context.Context, network, address string,
) (conn net.Conn, err error) {
	return d.DialContextHandler(
		ctx, tracing.ContextHandler(ctx), network, address,
	)
}

// DialContextHandler is like DialContext but we also optionally
// specify what handler is to be used.
func (d *Dialer) DialContextHandler(
	ctx context.Context, handler model.Handler, network, address string,
) (conn net.Conn, err error) {
	conn, _, _, err = d.DialContextEx(ctx, handler, network, address, false)
	if err != nil {
		// This is necessary because we're converting from
		// *measurement.Conn to net.Conn.
		return nil, err
	}
	return net.Conn(conn), nil
}

// DialContextEx is an extended DialContext where we may also
// optionally prevent processing of domain names.
func (d *Dialer) DialContextEx(
	ctx context.Context, handler model.Handler,
	network, address string, requireIP bool,
) (conn *connx.MeasuringConn, onlyhost, onlyport string, err error) {
	onlyhost, onlyport, err = net.SplitHostPort(address)
	if err != nil {
		return
	}
	connid := NextConnID()
	if net.ParseIP(onlyhost) != nil {
		conn, err = d.DialHostPort(
			ctx, handler, network, onlyhost, onlyport, connid,
		)
		return
	}
	if requireIP == true {
		err = errors.New("dialercontext: you passed me a domain name")
		return
	}
	start := time.Now()
	var addrs []string
	addrs, err = d.LookupHost(ctx, onlyhost)
	stop := time.Now()
	handler.OnMeasurement(model.Measurement{
		Resolve: &model.ResolveEvent{
			Addresses: addrs,
			ConnID:    connid,
			Duration:  stop.Sub(start),
			Error:     err,
			Hostname:  onlyhost,
			Time:      stop.Sub(d.dialer.Beginning),
		},
	})
	if err != nil {
		return
	}
	for _, addr := range addrs {
		conn, err = d.DialHostPort(ctx, handler, network, addr, onlyport, connid)
		if err == nil {
			return
		}
	}
	err = &net.OpError{
		Op:  "dial",
		Net: network,
		Err: errors.New("dialercontext: all connect attempts failed"),
	}
	return
}
