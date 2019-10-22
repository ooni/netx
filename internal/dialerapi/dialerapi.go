// Package dialerapi contains the dialer's API. The dialer defined
// in here implements basic DNS, but that is overridable.
package dialerapi

import (
	"context"
	"crypto/tls"
	"errors"
	"net"

	"github.com/ooni/netx/internal/connector"
	"github.com/ooni/netx/internal/connector/emittingconnector"
	"github.com/ooni/netx/internal/connector/ooconnector"
	"github.com/ooni/netx/internal/dnsclient/emittingdnsclient"
	"github.com/ooni/netx/internal/tlsconf"
	"github.com/ooni/netx/internal/tlshandshaker"
	"github.com/ooni/netx/internal/tlshandshaker/emittingtlshandshaker"
	"github.com/ooni/netx/internal/tlshandshaker/ootlshandshaker"
	"github.com/ooni/netx/internal/tracing"
)

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	Connector  connector.Model
	Handshaker tlshandshaker.Model
	LookupHost func(context.Context, string) ([]string, error)
	TLSConfig  *tls.Config
}

// NewDialer creates a new Dialer.
func NewDialer() *Dialer {
	return &Dialer{
		Connector:  emittingconnector.New(ooconnector.New()),
		Handshaker: emittingtlshandshaker.New(ootlshandshaker.New()),
		LookupHost: emittingdnsclient.New(&net.Resolver{
			// This is equivalent to ConfigureDNS("system", "...")
			PreferGo: true,
		}).LookupHost,
		TLSConfig: &tls.Config{},
	}
}

// DialContext is like Dial but the context allows to interrupt a
// pending connection attempt at any time.
func (d *Dialer) DialContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	return d.dialWithNewInfo(ctx, network, address, d.dialContext)
}

func (d *Dialer) dialContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	return d.flexibleDial(ctx, network, address, false)
}

func (d *Dialer) dialWithNewInfo(
	ctx context.Context, network, address string,
	dial func(context.Context, string, string) (net.Conn, error),
) (net.Conn, error) {
	// Because we're about to create a new connection, we need to have
	// a fresh context with everything pertaining to such connection
	if info := tracing.ContextInfo(ctx); info != nil {
		ctx = tracing.WithInfo(ctx, info.Clone("dialerapi.go"))
	}
	return dial(ctx, network, address)
}

// DialTLSContext dials a TLS connection with context
func (d *Dialer) DialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	return d.dialWithNewInfo(ctx, network, address, d.dialTLSContext)
}

func (d *Dialer) dialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	domain, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	conn, err := d.flexibleDial(ctx, network, address, false)
	if err != nil {
		return nil, err
	}
	tlsconn, err := d.Handshaker.Do(ctx, conn, d.TLSConfig, domain)
	if err != nil {
		conn.Close()
		return nil, err
	}
	// Note that we cannot wrap `tc` because the HTTP code assumes
	// a `*tls.Conn` when implementing ALPN.
	return tlsconn, err
}

func (d *Dialer) flexibleDial(
	ctx context.Context, network, address string, requireIP bool,
) (net.Conn, error) {
	onlyhost, onlyport, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if net.ParseIP(onlyhost) != nil {
		conn, err := d.Connector.DialContext(ctx, network, address)
		return conn, err
	}
	if requireIP == true {
		return nil, errors.New("dialerapi: you passed me a domain name")
	}
	var addrs []string
	addrs, err = d.LookupHost(ctx, onlyhost)
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		target := net.JoinHostPort(addr, onlyport)
		conn, err := d.Connector.DialContext(ctx, network, target)
		if err == nil {
			return conn, nil
		}
	}
	return nil, &net.OpError{
		Op:  "dial",
		Net: network,
		Err: errors.New("all connect attempts failed"),
	}
}

// SetCABundle configures the dialer to use a specific CA bundle.
func (d *Dialer) SetCABundle(path string) error {
	return tlsconf.SetCABundle(d.TLSConfig, path)
}

// ForceSpecificSNI forces using a specific SNI.
func (d *Dialer) ForceSpecificSNI(sni string) error {
	return tlsconf.ForceSpecificSNI(d.TLSConfig, sni)
}
