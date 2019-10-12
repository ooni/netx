// Package resolver contains code to create DNS resolvers.
package resolver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ooni/netx/dnsx"
	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/dialercontext"
	"github.com/ooni/netx/internal/dialerfull"
	"github.com/ooni/netx/internal/oodns"
	"github.com/ooni/netx/internal/tracing"
)

// ParseDNSConfigFromURL returns the network and address values you should pass
// to resolver.New on success, and error on failure.
func ParseDNSConfigFromURL(URL string) (network string, address string, err error) {
	parsed, err := url.Parse(URL)
	if err != nil {
		return "", "", err
	}
	if parsed.Scheme != "https" {
		return parsed.Scheme, parsed.Host, nil
	}
	return "doh", URL, nil
}

// New returns a new DNS resolver instance. The network and address arguments
// are the same of netx.ConfigureDNS.
func New(
	beginning time.Time, network, address string,
	newTransport func(beginning time.Time) http.RoundTripper,
) (dnsx.Client, error) {
	if network == "system" {
		return (&net.Resolver{PreferGo: false}), nil
	}
	if network == "netgo" {
		return (&net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				dialer := dialercontext.NewDialer(beginning)
				// TODO(bassosimone): this should actually be dialer.DialPacketConn
				// or something similar rather than exposing internals
				conn, _, _, err := dialer.DialContextEx(
					ctx, tracing.ContextHandler(ctx), network, address, false,
				)
				// convince Go this is really a net.PacketConn
				return &connx.DNSMeasuringConn{MeasuringConn: *conn}, err
			},
		}), nil
	}
	if network == "doh" {
		return oodns.NewClient(oodns.NewTransportDoH(&http.Client{
			Transport: newTransport(beginning),
		}, address)), nil
	}
	if network == "udp" {
		return oodns.NewClient(oodns.NewTransportUDP(
			address, (&defaultPortSetter{
				dialContext: dialercontext.NewDialer(beginning).DialContext,
				port:        "53",
			}).DialContext,
		)), nil
	}
	if network == "tcp" {
		return oodns.NewClient(oodns.NewTransportTCP(
			address, (&defaultPortSetter{
				dialContext: dialercontext.NewDialer(beginning).DialContext,
				port:        "53",
			}).DialContext,
		)), nil
	}
	if network == "dot" {
		return oodns.NewClient(oodns.NewTransportTCP(
			address, (&defaultPortSetter{
				dialContext: dialerfull.NewDialer(beginning).DialTLSContext,
				port:        "853",
			}).DialContext,
		)), nil
	}
	return nil, errors.New("resolver: not implemented")
}

type defaultPortSetter struct {
	dialContext func(
		ctx context.Context, network string, address string) (net.Conn, error)
	port string
}

func (dps *defaultPortSetter) DialContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	_, _, err := net.SplitHostPort(address)
	if err != nil && strings.HasSuffix(err.Error(), "missing port in address") {
		address = net.JoinHostPort(address, dps.port)
	}
	return dps.dialContext(ctx, network, address)
}
