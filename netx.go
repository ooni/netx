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
	"github.com/ooni/netx/model"
)

// Dialer performs measurements while dialing.
type Dialer struct {
	dialer *dialerapi.Dialer
}

// NewDialer returns a new Dialer instance.
func NewDialer(handler model.Handler) *Dialer {
	return &Dialer{
		dialer: dialerapi.NewDialer(time.Now(), handler),
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
// - "netgo": this indicates that Go should use its pure Go DNS
// resolver with the default server. The value of the address
// parameter is ignored when using "netgo". However, with this
// resolver we'll be able to see DNS packets.
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
//   d.SetResolver("system", "")
//   d.SetResolver("godns", "")
//   d.SetResolver("udp", "8.8.8.8:53")
//   d.SetResolver("tcp", "8.8.8.8:53")
//   d.SetResolver("dot", "dns.quad9.net")
//   d.SetResolver("doh", "https://cloudflare-dns.com/dns-query")
//
// ConfigureDNS is currently only executed when Go chooses to
// use the pure Go implementation of the DNS. This means that it
// does not work on Windows, where the C library is preferred. That
// is, on Windows you always use the "system" DNS.
func (d *Dialer) ConfigureDNS(network, address string) error {
	return dnsconf.ConfigureDNS(d.dialer, network, address)
}

// Dial creates a TCP or UDP connection. See net.Dial docs.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.dialer.Dial(network, address)
}

// DialContext is like Dial but the context allows to interrupt a
// pending connection attempt at any time.
func (d *Dialer) DialContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	return d.dialer.DialContext(ctx, network, address)
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, address string) (conn net.Conn, err error) {
	return d.dialer.DialTLS(network, address)
}

// NewResolver returns a new resolver using this Dialer as dialer for
// creating new network connections used for resolving. The arguments have
// the same meaning of ConfigureDNS. The returned resolver will not be
// used by this Dialer, however the network operations that it performs
// (e.g. creating a new connection) will use this Dialer. This is why
// NewResolver is a method rather than being just a free function.
//
// The Resolver returned by NewResolver shares the same limitation of
// ConfigureDNS. Under Windows the C library resolver is always used and
// therefore it is not possible for us to see DNS events.
func (d *Dialer) NewResolver(network, address string) (dnsx.Client, error) {
	return dnsconf.NewResolver(d.dialer, network, address)
}

// SetCABundle configures the dialer to use a specific CA bundle. This
// function is not goroutine safe. Make sure you call it befor starting
// to use this specific dialer.
func (d *Dialer) SetCABundle(path string) error {
	return d.dialer.SetCABundle(path)
}
