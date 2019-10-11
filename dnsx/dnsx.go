// Package dnsx contains OONI's DNS extensions
package dnsx

import (
	"context"
	"net"
)

// Client is a DNS client. The *net.Resolver used by Go implements
// this interface, but other implementations are possible.
type Client interface {
	// LookupAddr performs a reverse lookup of an address.
	LookupAddr(ctx context.Context, addr string) (names []string, err error)

	// LookupCNAME returns the canonical name of a given host.
	LookupCNAME(ctx context.Context, host string) (cname string, err error)

	// LookupHost resolves a hostname to a list of IP addresses.
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)

	// LookupMX resolves the DNS MX records for a given domain name.
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)

	// LookupNS resolves the DNS NS records for a given domain name.
	LookupNS(ctx context.Context, name string) ([]*net.NS, error)
}

// RoundTripper represent an abstract DNS transport.
type RoundTripper interface {
	// RoundTrip sends a DNS query and receives the reply.
	RoundTrip(query []byte) (reply []byte, err error)

	// RoundTripContext is like RoundTrip except that the context allows
	// to interrupt the pending operation at any moment.
	RoundTripContext(ctx context.Context, query []byte) (reply []byte, err error)
}
