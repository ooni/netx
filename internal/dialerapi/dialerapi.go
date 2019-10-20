// Package dialerapi contains the dialer's API. The dialer defined
// in here implements basic DNS, but that is overridable.
package dialerapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/internal/connector"
	"github.com/ooni/netx/internal/connector/emittingconnector"
	"github.com/ooni/netx/internal/connector/ooconnector"
	"github.com/ooni/netx/internal/dnsclient/emittingdnsclient"
	"github.com/ooni/netx/internal/tlshandshaker"
	"github.com/ooni/netx/internal/tlshandshaker/emittingtlshandshaker"
	"github.com/ooni/netx/internal/tlshandshaker/ootlshandshaker"
	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
)

var nextConnID int64

func getNextConnID() int64 {
	return atomic.AddInt64(&nextConnID, 1)
}

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	Beginning  time.Time
	Connector  connector.Model
	Handler    model.Handler
	Handshaker tlshandshaker.Model
	LookupHost func(context.Context, string) ([]string, error)
	TLSConfig  *tls.Config
}

// NewDialer creates a new Dialer.
func NewDialer(beginning time.Time, handler model.Handler) *Dialer {
	return &Dialer{
		Beginning:  beginning,
		Connector:  emittingconnector.New(ooconnector.New()),
		Handler:    handler,
		Handshaker: emittingtlshandshaker.New(ootlshandshaker.New()),
		LookupHost: emittingdnsclient.New(&net.Resolver{
			// This is equivalent to ConfigureDNS("system", "...")
			PreferGo: true,
		}).LookupHost,
		TLSConfig: &tls.Config{},
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
) (net.Conn, error) {
	if info := tracing.ContextInfo(ctx); info == nil {
		ctx = tracing.WithInfo(ctx, &tracing.Info{
			Beginning: d.Beginning,
			Handler:   d.Handler,
		})
	}
	return d.flexibleDial(ctx, network, address, false)
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, address string) (net.Conn, error) {
	return d.DialTLSContext(context.Background(), network, address)
}

// DialTLSContext is like DialTLS but with context.
func (d *Dialer) DialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	domain, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if info := tracing.ContextInfo(ctx); info == nil {
		ctx = tracing.WithInfo(ctx, &tracing.Info{
			Beginning: d.Beginning,
			Handler:   d.Handler,
		})
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
	if info := tracing.ContextInfo(ctx); info != nil {
		info.ConnID = getNextConnID()
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
	cert, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(cert)
	d.TLSConfig.RootCAs = pool
	return nil
}

// ForceSpecificSNI forces using a specific SNI.
func (d *Dialer) ForceSpecificSNI(sni string) error {
	d.TLSConfig.ServerName = sni
	return nil
}
