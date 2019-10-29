// Package resolver contains code to create a resolver
package resolver

import (
	"net/http"

	"github.com/ooni/netx/internal/resolver/dnstransport/dnsoverhttps"
	"github.com/ooni/netx/internal/resolver/dnstransport/dnsovertcp"
	"github.com/ooni/netx/internal/resolver/dnstransport/dnsoverudp"
	"github.com/ooni/netx/internal/resolver/ooniresolver"
	"github.com/ooni/netx/model"
)

// NewResolverUDP creates a new UDP resolver.
func NewResolverUDP(dialer model.Dialer, address string) *ooniresolver.Resolver {
	return ooniresolver.New(dnsoverudp.NewTransport(dialer, address))
}

// NewResolverTCP creates a new TCP resolver.
func NewResolverTCP(dialer model.Dialer, address string) *ooniresolver.Resolver {
	return ooniresolver.New(dnsovertcp.NewTransportTCP(dialer, address))
}

// NewResolverTLS creates a new DoT resolver.
func NewResolverTLS(dialer model.TLSDialer, address string) *ooniresolver.Resolver {
	return ooniresolver.New(dnsovertcp.NewTransportTLS(dialer, address))
}

// NewResolverHTTPS creates a new DoH resolver.
func NewResolverHTTPS(client *http.Client, address string) *ooniresolver.Resolver {
	return ooniresolver.New(dnsoverhttps.NewTransport(client, address))
}
