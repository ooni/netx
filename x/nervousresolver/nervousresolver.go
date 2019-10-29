// Package nervousresolver contains OONI's nervous resolver
// that reacts to errors and performs actions.
//
// This package is still experimental. See LookupHost docs for
// an overview of what we're doing here.
package nervousresolver

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/m-lab/go/rtx"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal"
	"github.com/ooni/netx/internal/transactionid"
	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/nervousresolver/bogon"
	"github.com/ooni/netx/x/scoreboard"
)

// Resolver is OONI's nervous resolver.
type Resolver struct {
	bogonsCount int64
	primary     model.DNSResolver
	secondary   model.DNSResolver
}

// New creates a new OONI nervous resolver instance.
func New(primary, secondary model.DNSResolver) *Resolver {
	return &Resolver{
		primary:   primary,
		secondary: secondary,
	}
}

// LookupAddr returns the name of the provided IP address
func (c *Resolver) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	return c.primary.LookupAddr(ctx, addr)
}

// LookupCNAME returns the canonical name of a host
func (c *Resolver) LookupCNAME(ctx context.Context, host string) (cname string, err error) {
	return c.primary.LookupCNAME(ctx, host)
}

type bogonLookup struct {
	Addresses []string
	Comment   string
	Hostname  string
}

// LookupHost returns the IP addresses of a host.
//
// This code in particular checks whether the first DNS reply is
// reasonable and, if not, it will query a secondary resolver.
//
// The general idea here is that the first resolver is hopefully
// getaddrinfo and the secondary resolver is DoH/DoT.
//
// The code in here is an initial, experimental implementation of a
// design document on which we're working with Vinicius Fortuna,
// Jigsaw, aimed at significantly improving OONI measurements quality.
//
// TODO(bassosimone): integrate more ideas from the design doc.
func (c *Resolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := c.primary.LookupHost(ctx, hostname)
	if err == nil {
		for _, addr := range addrs {
			if bogon.Check(addr) == true {
				return c.detectedBogon(ctx, hostname, addrs)
			}
		}
	}
	return addrs, err
}

func (c *Resolver) detectedBogon(
	ctx context.Context, hostname string, addrs []string,
) ([]string, error) {
	atomic.AddInt64(&c.bogonsCount, 1)
	root := model.ContextMeasurementRootOrDefault(ctx)
	durationSinceBeginning := time.Now().Sub(root.Beginning)
	root.X.Scoreboard.AddDNSBogonInfo(scoreboard.DNSBogonInfo{
		Addresses:              addrs,
		DurationSinceBeginning: durationSinceBeginning,
		Domain:                 hostname,
		FallbackPlan:           "ignore_and_retry_with_doh",
	})
	value := bogonLookup{
		Addresses: addrs,
		Comment:   "detected bogon DNS reply; retry using DoH resolver",
		Hostname:  hostname,
	}
	// TODO(bassosimone): because this is a PoC, I'm using the
	// extension event model. I believe there should be a specific
	// first class event emitted when we see a bogon, tho.
	root.Handler.OnMeasurement(model.Measurement{
		Extension: &model.ExtensionEvent{
			DurationSinceBeginning: durationSinceBeginning,
			Key:                    fmt.Sprintf("%T", value),
			Severity:               "WARN",
			TransactionID:          transactionid.ContextTransactionID(ctx),
			Value:                  value,
		},
	})
	return c.secondary.LookupHost(ctx, hostname)
}

// LookupMX returns the MX records of a specific name
func (c *Resolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return c.primary.LookupMX(ctx, name)
}

// LookupNS returns the NS records of a specific name
func (c *Resolver) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	return c.primary.LookupNS(ctx, name)
}

// Default is the default nervous resolver
var Default *Resolver

func init() {
	system, err := internal.NewResolver(
		time.Time{}, handlers.NoHandler,
		"system", "",
	)
	rtx.PanicOnError(err, "internal.NewResolver #1 failed")
	// TODO(bassosimone): because this is a PoC, I'm using for
	// now the address of Cloudflare. We should probably configure
	// this when integrating in probe-engine.
	overhttps, err := internal.NewResolver(
		time.Time{}, handlers.NoHandler,
		"doh", "https://cloudflare-dns.com/dns-query",
	)
	rtx.PanicOnError(err, "internal.NewResolver #2 failed")
	Default = New(system, overhttps)
}
