// Package internal contains internal code.
package internal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ooni/netx/internal/dialer"
	"github.com/ooni/netx/internal/httptransport"
	"github.com/ooni/netx/internal/resolver"
	"github.com/ooni/netx/model"
	"golang.org/x/net/http2"
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
		Resolver:  resolver.NewResolverSystem(),
		TLSConfig: new(tls.Config),
	}
}

// Dial creates a TCP or UDP connection. See net.Dial docs.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func maybeWithMeasurementRoot(
	ctx context.Context, beginning time.Time, handler model.Handler,
) context.Context {
	if model.ContextMeasurementRoot(ctx) != nil {
		return ctx
	}
	return model.WithMeasurementRoot(ctx, &model.MeasurementRoot{
		Beginning: beginning,
		Handler:   handler,
	})
}

// DialContext is like Dial but the context allows to interrupt a
// pending connection attempt at any time.
func (d *Dialer) DialContext(
	ctx context.Context, network, address string,
) (conn net.Conn, err error) {
	ctx = maybeWithMeasurementRoot(ctx, d.Beginning, d.Handler)
	return dialer.New(
		d.Resolver, new(net.Dialer),
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
	ctx = maybeWithMeasurementRoot(ctx, d.Beginning, d.Handler)
	return dialer.NewTLS(
		dialer.New(d.Resolver, new(net.Dialer)),
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

// ConfigureDNS implements netx.Dialer.ConfigureDNS.
func (d *Dialer) ConfigureDNS(network, address string) error {
	r, err := NewResolver(d.Beginning, d.Handler, network, address)
	if err == nil {
		d.Resolver = r
	}
	return err
}

// SetResolver implements netx.Dialer.SetResolver.
func (d *Dialer) SetResolver(r model.DNSResolver) {
	d.Resolver = r
}

func newHTTPClientForDoH(beginning time.Time, handler model.Handler) *http.Client {
	transport := NewHTTPTransport(beginning, handler, NewDialer(beginning, handler))
	return &http.Client{Transport: transport}
}

func withPort(address, port string) string {
	// Handle the case where port was not specified. We have written in
	// a bunch of places that we can just pass a domain in this case and
	// so we need to gracefully ensure this is still possible.
	_, _, err := net.SplitHostPort(address)
	if err != nil && strings.Contains(err.Error(), "missing port in address") {
		address = net.JoinHostPort(address, port)
	}
	return address
}

type resolverWrapper struct {
	beginning time.Time
	handler   model.Handler
	resolver  model.DNSResolver
}

func newResolverWrapper(
	beginning time.Time, handler model.Handler,
	resolver model.DNSResolver,
) *resolverWrapper {
	return &resolverWrapper{
		beginning: beginning,
		handler:   handler,
		resolver:  resolver,
	}
}

// LookupAddr returns the name of the provided IP address
func (r *resolverWrapper) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	ctx = maybeWithMeasurementRoot(ctx, r.beginning, r.handler)
	return r.resolver.LookupAddr(ctx, addr)
}

// LookupCNAME returns the canonical name of a host
func (r *resolverWrapper) LookupCNAME(ctx context.Context, host string) (string, error) {
	ctx = maybeWithMeasurementRoot(ctx, r.beginning, r.handler)
	return r.resolver.LookupCNAME(ctx, host)
}

// LookupHost returns the IP addresses of a host
func (r *resolverWrapper) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	ctx = maybeWithMeasurementRoot(ctx, r.beginning, r.handler)
	return r.resolver.LookupHost(ctx, hostname)
}

// LookupMX returns the MX records of a specific name
func (r *resolverWrapper) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	ctx = maybeWithMeasurementRoot(ctx, r.beginning, r.handler)
	return r.resolver.LookupMX(ctx, name)
}

// LookupNS returns the NS records of a specific name
func (r *resolverWrapper) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	ctx = maybeWithMeasurementRoot(ctx, r.beginning, r.handler)
	return r.resolver.LookupNS(ctx, name)
}

// NewResolver returns a new resolver
func NewResolver(
	beginning time.Time, handler model.Handler, network, address string,
) (model.DNSResolver, error) {
	// Implementation note: system need to be dealt with
	// separately because it doesn't have any transport.
	if network == "system" {
		return newResolverWrapper(
			beginning, handler, resolver.NewResolverSystem()), nil
	}
	if network == "doh" {
		return newResolverWrapper(beginning, handler, resolver.NewResolverHTTPS(
			newHTTPClientForDoH(beginning, handler), address,
		)), nil
	}
	if network == "dot" {
		// We need a child dialer here to avoid an endless loop where the
		// dialer will ask us to resolve, we'll tell the dialer to dial, it
		// will ask us to resolve, ...
		return newResolverWrapper(beginning, handler, resolver.NewResolverTLS(
			NewDialer(beginning, handler), withPort(address, "853"),
		)), nil
	}
	if network == "tcp" {
		// Same rationale as above: avoid possible endless loop
		return newResolverWrapper(beginning, handler, resolver.NewResolverTCP(
			NewDialer(beginning, handler), withPort(address, "53"),
		)), nil
	}
	if network == "udp" {
		// Same rationale as above: avoid possible endless loop
		return newResolverWrapper(beginning, handler, resolver.NewResolverUDP(
			NewDialer(beginning, handler), withPort(address, "53"),
		)), nil
	}
	return nil, errors.New("resolver.New: unsupported network value")
}

// HTTPTransport performs single HTTP transactions and emits
// measurement events as they happen.
type HTTPTransport struct {
	Transport    *http.Transport
	Handler      model.Handler
	Beginning    time.Time
	roundTripper http.RoundTripper
}

// NewHTTPTransport creates a new Transport.
func NewHTTPTransport(
	beginning time.Time,
	handler model.Handler,
	dialer *Dialer,
) *HTTPTransport {
	baseTransport := &http.Transport{
		// The following values are copied from Go 1.12 docs and match
		// what should be used by the default transport
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   10 * time.Second,
	}
	ooniTransport := httptransport.New(baseTransport)
	// Configure h2 and make sure that the custom TLSConfig we use for dialing
	// is actually compatible with upgrading to h2. (This mainly means we
	// need to make sure we include "h2" in the NextProtos array.) Because
	// http2.ConfigureTransport only returns error when we have already
	// configured http2, it is safe to ignore the return value.
	http2.ConfigureTransport(baseTransport)
	// Since we're not going to use our dialer for TLS, the main purpose of
	// the following line is to make sure ForseSpecificSNI has impact on the
	// config we are going to use when doing TLS. The code is as such since
	// we used to force net/http through using dialer.DialTLS.
	dialer.TLSConfig = baseTransport.TLSClientConfig
	// Arrange the configuration such that we always use `dialer` for dialing
	// cleartext connections. The net/http code will dial TLS connections.
	baseTransport.DialContext = dialer.DialContext
	// Better for Cloudflare DNS and also better because we have less
	// noisy events and we can better understand what happened.
	baseTransport.MaxConnsPerHost = 1
	// The following (1) reduces the number of headers that Go will
	// automatically send for us and (2) ensures that we always receive
	// back the true headers, such as Content-Length. This change is
	// functional to OONI's goal of observing the network.
	baseTransport.DisableCompression = true
	return &HTTPTransport{
		Transport:    baseTransport,
		Handler:      handler,
		Beginning:    beginning,
		roundTripper: ooniTransport,
	}
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *HTTPTransport) RoundTrip(
	req *http.Request,
) (resp *http.Response, err error) {
	ctx := maybeWithMeasurementRoot(req.Context(), t.Beginning, t.Handler)
	req = req.WithContext(ctx)
	return t.roundTripper.RoundTrip(req)
}

// CloseIdleConnections closes the idle connections.
func (t *HTTPTransport) CloseIdleConnections() {
	// Adapted from net/http code
	type closeIdler interface {
		CloseIdleConnections()
	}
	if tr, ok := t.roundTripper.(closeIdler); ok {
		tr.CloseIdleConnections()
	}
}
