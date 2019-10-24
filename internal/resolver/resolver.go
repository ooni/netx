// Package resolver contains code to create a resolver
package resolver

import (
	"net/http"
	"time"

	"github.com/ooni/netx/internal/resolver/dnstransport/dnsoverhttps"
	"github.com/ooni/netx/internal/resolver/dnstransport/dnsovertcp"
	"github.com/ooni/netx/internal/resolver/dnstransport/dnsoverudp"
	"github.com/ooni/netx/internal/resolver/ooniresolver"
	"github.com/ooni/netx/model"
)

// NewResolverUDP creates a new UDP resolver.
func NewResolverUDP(
	beginning time.Time, handler model.Handler,
	dialer model.Dialer, address string,
) *ooniresolver.Resolver {
	return ooniresolver.New(
		beginning, handler, dnsoverudp.NewTransport(dialer, address),
	)
}

// NewResolverTCP creates a new TCP resolver.
func NewResolverTCP(
	beginning time.Time, handler model.Handler,
	dialer model.Dialer, address string,
) *ooniresolver.Resolver {
	return ooniresolver.New(
		beginning, handler, dnsovertcp.NewTransport(dialer, address),
	)
}

// NewResolverTLS creates a new DoT resolver.
func NewResolverTLS(
	beginning time.Time, handler model.Handler,
	dialer model.TLSDialer, address string,
) *ooniresolver.Resolver {
	return ooniresolver.New(
		beginning, handler, dnsovertcp.NewTransport(
			dnsovertcp.NewTLSDialerAdapter(dialer),
			address,
		),
	)
}

// NewResolverHTTPS creates a new DoH resolver.
func NewResolverHTTPS(
	beginning time.Time, handler model.Handler,
	client *http.Client, address string,
) *ooniresolver.Resolver {
	return ooniresolver.New(
		beginning, handler, dnsoverhttps.NewTransport(
			client, address,
		),
	)
}
