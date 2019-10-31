// Package emitterresolver contains the resolver that emits events
package emitterresolver

import (
	"context"
	"net"
	"time"

	"github.com/ooni/netx/internal/dialid"
	"github.com/ooni/netx/internal/errwrapper"
	"github.com/ooni/netx/internal/transactionid"
	"github.com/ooni/netx/model"
)

// Resolver is the emitter resolver
type Resolver struct {
	resolver model.DNSResolver
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
	addrs, err := r.resolver.LookupHost(ctx, hostname)
	err = errwrapper.SafeErrWrapperBuilder{
		DialID:        dialID,
		Error:         err,
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
	return addrs, err
}

// LookupMX returns the MX records of a specific name
func (r *Resolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return r.resolver.LookupMX(ctx, name)
}

// LookupNS returns the NS records of a specific name
func (r *Resolver) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	return r.resolver.LookupNS(ctx, name)
}
