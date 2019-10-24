// Package internal contains internal code.
package internal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"time"

	"github.com/ooni/netx/internal/dialer"
	"github.com/ooni/netx/model"
)

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	Beginning time.Time
	Handler   model.Handler
	Resolver  model.DNSResolver
	TLSConfig *tls.Config
}

// NewDialer creates a new Dialer.
func NewDialer(
	beginning time.Time, handler model.Handler,
) (d *Dialer) {
	return &Dialer{
		Beginning: beginning,
		Handler:   handler,
		Resolver:  new(net.Resolver),
		TLSConfig: new(tls.Config),
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
	return dialer.New(
		d.Beginning, d.Handler, d.Resolver, new(net.Dialer),
	).DialContext(ctx, network, address)
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, address string) (net.Conn, error) {
	ctx := context.Background()
	return d.DialTLSContext(ctx, network, address)
}

// DialTLSContext is like DialTLS, but with context
func (d *Dialer) DialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	return dialer.NewTLS(
		d.Beginning,
		d.Handler,
		dialer.New(
			d.Beginning,
			d.Handler,
			d.Resolver,
			new(net.Dialer),
		),
		d.TLSConfig,
	).DialTLSContext(ctx, network, address)
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
