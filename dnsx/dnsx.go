// Package dnsx contains OONI's DNS extensions
package dnsx

import (
	"context"
	"net"
)

// Resolver is a DNS resolver.
type Resolver interface {
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
