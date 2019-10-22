// Package netx contains OONI's net extensions.
//
// This package provides a replacement for net.Dialer that can Dial,
// DialContext, and DialTLS. During its lifecycle this modified Dialer
// will emit network level events on a channel.
package netx

import (
	"context"
	"net"
	"time"

	"github.com/ooni/netx/dnsx"
	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/internal/dnsconf"
	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
)

// Dialer performs measurements while dialing.
type Dialer struct {
	Beginning time.Time
	Handler   model.Handler
	dialer    *dialerapi.Dialer
}

// NewDialer returns a new Dialer instance.
func NewDialer(handler model.Handler) *Dialer {
	return &Dialer{
		Beginning: time.Now(),
		Handler:   handler,
		dialer:    dialerapi.NewDialer(),
	}
}

// ConfigureDNS configures the DNS resolver. The network argument
// selects the type of resolver. The address argument indicates the
// resolver address and depends on the network.
//
// This functionality is not goroutine safe. You should only change
// the DNS settings before starting to use the Dialer.
//
// The following is a list of all the possible network values:
//
// - "system": this indicates that Go should use the system resolver
// and prevents us from seeing any DNS packet. The value of the
// address parameter is ignored when using "system". If you do
// not ConfigureDNS, this is the default resolver used.
//
// - "udp": indicates that we should send queries using UDP. In this
// case the address is a host, port UDP endpoint.
//
// - "tcp": like "udp" but we use TCP.
//
// - "dot": we use DNS over TLS (DoT). In this case the address is
// the domain name of the DoT server.
//
// - "doh": we use DNS over HTTPS (DoH). In this case the address is
// the URL of the DoH server.
//
// For example:
//
//   d.ConfigureDNS("system", "")
//   d.ConfigureDNS("udp", "8.8.8.8:53")
//   d.ConfigureDNS("tcp", "8.8.8.8:53")
//   d.ConfigureDNS("dot", "dns.quad9.net")
//   d.ConfigureDNS("doh", "https://cloudflare-dns.com/dns-query")
func (d *Dialer) ConfigureDNS(network, address string) error {
	return dnsconf.ConfigureDNS(d.dialer, network, address)
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
	// Setup tracing with current handler and start time
	ctx = tracing.WithInfo(ctx, &tracing.Info{
		Beginning: d.Beginning,
		Handler:   d.Handler,
	})
	return d.dialer.DialContext(ctx, network, address)
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, address string) (conn net.Conn, err error) {
	// Setup tracing with current handler and start time
	ctx := tracing.WithInfo(context.Background(), &tracing.Info{
		Beginning: d.Beginning,
		Handler:   d.Handler,
	})
	return d.dialer.DialTLSContext(ctx, network, address)
}

// NewResolver returns a new resolver using this Dialer as dialer for
// creating new network connections used for resolving. The arguments have
// the same meaning of ConfigureDNS. The returned resolver will not be
// used by this Dialer, however the network operations that it performs
// (e.g. creating a new connection) will use this Dialer. This is why
// NewResolver is a method rather than being just a free function.
func (d *Dialer) NewResolver(network, address string) (dnsx.Client, error) {
	resolver, err := dnsconf.NewResolver(network, address)
	if err == nil {
		resolver = &resolverWrapper{
			beginning: d.Beginning,
			handler:   d.Handler,
			resolver:  resolver,
		}
	}
	return resolver, err
}

// SetCABundle configures the dialer to use a specific CA bundle. This
// function is not goroutine safe. Make sure you call it before starting
// to use this specific dialer.
func (d *Dialer) SetCABundle(path string) error {
	return d.dialer.SetCABundle(path)
}

// ForceSpecificSNI forces using a specific SNI.
func (d *Dialer) ForceSpecificSNI(sni string) error {
	return d.dialer.ForceSpecificSNI(sni)
}

type resolverWrapper struct {
	beginning time.Time
	handler   model.Handler
	resolver  dnsx.Client
}

// LookupAddr performs a reverse lookup of an address.
func (r *resolverWrapper) LookupAddr(
	ctx context.Context, addr string,
) (names []string, err error) {
	// Setup tracing with current handler and start time
	ctx = tracing.WithInfo(ctx, &tracing.Info{
		Beginning: r.beginning,
		Handler:   r.handler,
	})
	return r.resolver.LookupAddr(ctx, addr)
}

// LookupCNAME returns the canonical name of a given host.
func (r *resolverWrapper) LookupCNAME(
	ctx context.Context, host string,
) (cname string, err error) {
	// Setup tracing with current handler and start time
	ctx = tracing.WithInfo(ctx, &tracing.Info{
		Beginning: r.beginning,
		Handler:   r.handler,
	})
	return r.resolver.LookupCNAME(ctx, host)
}

// LookupHost resolves a hostname to a list of IP addresses.
func (r *resolverWrapper) LookupHost(
	ctx context.Context, hostname string,
) (addrs []string, err error) {
	// Setup tracing with current handler and start time
	ctx = tracing.WithInfo(ctx, &tracing.Info{
		Beginning: r.beginning,
		Handler:   r.handler,
	})
	return r.resolver.LookupHost(ctx, hostname)
}

// LookupMX resolves the DNS MX records for a given domain name.
func (r *resolverWrapper) LookupMX(
	ctx context.Context, name string,
) ([]*net.MX, error) {
	// Setup tracing with current handler and start time
	ctx = tracing.WithInfo(ctx, &tracing.Info{
		Beginning: r.beginning,
		Handler:   r.handler,
	})
	return r.resolver.LookupMX(ctx, name)
}

// LookupNS resolves the DNS NS records for a given domain name.
func (r *resolverWrapper) LookupNS(
	ctx context.Context, name string,
) ([]*net.NS, error) {
	// Setup tracing with current handler and start time
	ctx = tracing.WithInfo(ctx, &tracing.Info{
		Beginning: r.beginning,
		Handler:   r.handler,
	})
	return r.resolver.LookupNS(ctx, name)
}
