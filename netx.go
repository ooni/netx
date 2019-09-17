// Package netx contains OONI's net extensions.
//
// This package provides a replacement for net.Dialer that can Dial,
// DialContext, and DialTLS. During its lifecycle this modified Dialer
// will observe network level events and collect Measurements.
package netx

import (
	"context"
	"net"
	"time"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dnsconf"
	"github.com/bassosimone/netx/model"
)

// Dialer performs measurements while dialing.
type Dialer struct {
	dialer *dialerapi.Dialer
}

// NewDialer returns a new Dialer instance.
func NewDialer(ch chan model.Measurement) *Dialer {
	return &Dialer{
		dialer: dialerapi.NewDialer(time.Now(), ch),
	}
}

// ConfigureDNS configures the DNS resolver. The |network| argument
// selects the type of resolver. The |address| argument indicates the
// resolver address and depends on the |network|. The following is a
// list of all the possible |network| values:
//
// - "udp": indicates that we should send queries using UDP. In this
// case the |address| is a host, port UDP endpoint.
//
// - "tcp": like UDP but we use DNS over TCP.
//
// - "dot": like TCP but we use DNS over TLS (DoT). In this case the
// |address| is the domain name of the DoT server.
//
// - "doh": we use DNS over HTTPS. In this case the |address| is
// the full URL to be used as the resolver.
//
// Examples
//
//   d.SetResolver("udp", "8.8.8.8:53")
//   d.SetResolver("tcp", "8.8.8.8:53")
//   d.SetResolver("dot", "dns.quad9.net")
//   d.SetResolver("doh", "https://cloudflare-dns.com/dns-query")
//
// Bugs
//
// This modified DNS code is currently only executed when Go chooses
// to use the pure Go implementation of the DNS. This means that it
// should not be working on Windows, where the C library is preferred.
func (d *Dialer) ConfigureDNS(network, address string) error {
	return dnsconf.Do(d.dialer, network, address)
}

// Dial creates a TCP or UDP connection. See net.Dial docs.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.dialer.Dial(network, address)
}

// DialContext is like Dial but the context allows to interrupt a
// pending connection attempt at any time.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.dialer.DialContext(ctx, network, address)
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, address string) (conn net.Conn, err error) {
	return d.dialer.DialTLS(network, address)
}
