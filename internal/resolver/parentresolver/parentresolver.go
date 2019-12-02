// Package parentresolver contains the parent resolver
package parentresolver

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/internal/dialid"
	"github.com/ooni/netx/internal/errwrapper"
	"github.com/ooni/netx/internal/resolver/bogondetector"
	"github.com/ooni/netx/internal/transactionid"
	"github.com/ooni/netx/modelx"
	"github.com/ooni/netx/x/scoreboard"
)

// Resolver is the emitter resolver
type Resolver struct {
	bogonsCount int64
	resolver    modelx.DNSResolver
}

// New creates a new emitter resolver
func New(resolver modelx.DNSResolver) *Resolver {
	return &Resolver{resolver: resolver}
}

// LookupAddr returns the name of the provided IP address
func (r *Resolver) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	return r.resolver.LookupAddr(ctx, addr)
}

// LookupCNAME returns the canonical name of a host
func (r *Resolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	return r.resolver.LookupCNAME(ctx, host)
}

type queryableTransport interface {
	Network() string
	Address() string
}

type queryableResolver interface {
	Transport() modelx.DNSRoundTripper
}

func (r *Resolver) queryTransport() (network string, address string) {
	if reso, okay := r.resolver.(queryableResolver); okay {
		if transport, okay := reso.Transport().(queryableTransport); okay {
			network, address = transport.Network(), transport.Address()
		}
	}
	return
}

// LookupHost returns the IP addresses of a host
func (r *Resolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	network, address := r.queryTransport()
	dialID := dialid.ContextDialID(ctx)
	txID := transactionid.ContextTransactionID(ctx)
	root := modelx.ContextMeasurementRootOrDefault(ctx)
	root.Handler.OnMeasurement(modelx.Measurement{
		ResolveStart: &modelx.ResolveStartEvent{
			DialID:                 dialID,
			DurationSinceBeginning: time.Now().Sub(root.Beginning),
			Hostname:               hostname,
			TransactionID:          txID,
			TransportAddress:       address,
			TransportNetwork:       network,
		},
	})
	addrs, err := r.lookupHost(ctx, hostname)
	err = errwrapper.SafeErrWrapperBuilder{
		DialID:        dialID,
		Error:         err,
		Operation:     "resolve",
		TransactionID: txID,
	}.MaybeBuild()
	root.Handler.OnMeasurement(modelx.Measurement{
		ResolveDone: &modelx.ResolveDoneEvent{
			Addresses:              addrs,
			DialID:                 dialID,
			DurationSinceBeginning: time.Now().Sub(root.Beginning),
			Error:                  err,
			Hostname:               hostname,
			TransactionID:          txID,
			TransportAddress:       address,
			TransportNetwork:       network,
		},
	})
	// Respect general Go expectation that one doesn't return
	// both a value and a non-nil error
	if errors.Is(err, modelx.ErrDNSBogon) {
		addrs = nil
	}
	return addrs, err
}

func (r *Resolver) lookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := r.resolver.LookupHost(ctx, hostname)
	for _, addr := range addrs {
		if bogondetector.Check(addr) == true {
			return r.detectedBogon(ctx, hostname, addrs)
		}
	}
	return addrs, err
}

func (r *Resolver) detectedBogon(
	ctx context.Context, hostname string, addrs []string,
) ([]string, error) {
	atomic.AddInt64(&r.bogonsCount, 1)
	root := modelx.ContextMeasurementRootOrDefault(ctx)
	durationSinceBeginning := time.Now().Sub(root.Beginning)
	root.X.Scoreboard.AddDNSBogonInfo(scoreboard.DNSBogonInfo{
		Addresses:              addrs,
		DurationSinceBeginning: durationSinceBeginning,
		Domain:                 hostname,
		FallbackPlan:           "let_caller_decide",
	})
	// Note that here we return root.ErrDNSBogon, which by default
	// is nil, meaning that we'll not treat the bogon as hard error
	// but we'll register it in the scoreboard. The caller should
	// ensure that we won't return a value and an error at the same
	// time. See issue <https://github.com/ooni/netx/issues/126> for
	// more on why by default a bogon does not cause an error.
	return addrs, root.ErrDNSBogon
}

// LookupMX returns the MX records of a specific name
func (r *Resolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return r.resolver.LookupMX(ctx, name)
}

// LookupNS returns the NS records of a specific name
func (r *Resolver) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	return r.resolver.LookupNS(ctx, name)
}
