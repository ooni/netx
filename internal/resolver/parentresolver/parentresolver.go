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
	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/scoreboard"
)

// Resolver is the emitter resolver
type Resolver struct {
	bogonsCount int64
	resolver    model.DNSResolver
}

// New creates a new emitter resolver
func New(resolver model.DNSResolver) *Resolver {
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
	Transport() model.DNSRoundTripper
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
	root := model.ContextMeasurementRootOrDefault(ctx)
	root.Handler.OnMeasurement(model.Measurement{
		ResolveStart: &model.ResolveStartEvent{
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
	root.Handler.OnMeasurement(model.Measurement{
		ResolveDone: &model.ResolveDoneEvent{
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
	if errors.Is(err, errwrapper.ErrDNSBogon) {
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
	root := model.ContextMeasurementRootOrDefault(ctx)
	durationSinceBeginning := time.Now().Sub(root.Beginning)
	root.X.Scoreboard.AddDNSBogonInfo(scoreboard.DNSBogonInfo{
		Addresses:              addrs,
		DurationSinceBeginning: durationSinceBeginning,
		Domain:                 hostname,
		FallbackPlan:           "let_caller_decide",
	})
	// We're returning non nil addrs so the caller logs it
	// but the caller is assumed to not return addrs
	return addrs, errwrapper.ErrDNSBogon
}

// LookupMX returns the MX records of a specific name
func (r *Resolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return r.resolver.LookupMX(ctx, name)
}

// LookupNS returns the NS records of a specific name
func (r *Resolver) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	return r.resolver.LookupNS(ctx, name)
}
