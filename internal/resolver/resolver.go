// Package resolver contains code to create a resolver
package resolver

import (
	"net"
	"net/http"

	"github.com/ooni/netx/internal/resolver/dnstransport/dnsoverhttps"
	"github.com/ooni/netx/internal/resolver/dnstransport/dnsovertcp"
	"github.com/ooni/netx/internal/resolver/dnstransport/dnsoverudp"
	"github.com/ooni/netx/internal/resolver/emitterresolver"
	"github.com/ooni/netx/internal/resolver/ooniresolver"
	"github.com/ooni/netx/internal/resolver/systemresolver"
	"github.com/ooni/netx/model"
)

// NewResolverSystem creates a new Go/system resolver.
func NewResolverSystem() *emitterresolver.Resolver {
	return emitterresolver.New(systemresolver.New(new(net.Resolver)))
}

// NewResolverUDP creates a new UDP resolver.
func NewResolverUDP(dialer model.Dialer, address string) *emitterresolver.Resolver {
	return emitterresolver.New(
		ooniresolver.New(dnsoverudp.NewTransport(dialer, address)),
	)
}

// NewResolverTCP creates a new TCP resolver.
func NewResolverTCP(dialer model.Dialer, address string) *emitterresolver.Resolver {
	return emitterresolver.New(
		ooniresolver.New(dnsovertcp.NewTransportTCP(dialer, address)),
	)
}

// NewResolverTLS creates a new DoT resolver.
func NewResolverTLS(dialer model.TLSDialer, address string) *emitterresolver.Resolver {
	return emitterresolver.New(
		ooniresolver.New(dnsovertcp.NewTransportTLS(dialer, address)),
	)
}

// NewResolverHTTPS creates a new DoH resolver.
func NewResolverHTTPS(client *http.Client, address string) *emitterresolver.Resolver {
	return emitterresolver.New(
		ooniresolver.New(dnsoverhttps.NewTransport(client, address)),
	)
}
