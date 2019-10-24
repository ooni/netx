// Package resolver contains the resolver
package resolver

import (
	"context"
	"time"

	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/dialid"
)

// Resolver is a resolver
type Resolver struct {
	beginning time.Time
	handler   model.Handler
	resolver  model.Resolver
}

// New creates a new resolver
func New(
	beginning time.Time,
	handler model.Handler,
	resolver model.Resolver,
) *Resolver {
	return &Resolver{
		beginning: beginning,
		handler:   handler,
		resolver:  resolver,
	}
}

// LookupHost resolves a specific hostname
func (r *Resolver) LookupHost(
	ctx context.Context, hostname string,
) ([]string, error) {
	start := time.Now()
	addrs, err := r.resolver.LookupHost(ctx, hostname)
	stop := time.Now()
	m := model.Measurement{
		Resolve: &model.ResolveEvent{
			Addresses: addrs,
			DialID:    dialid.ContextDialID(ctx),
			Duration:  stop.Sub(start),
			Error:     err,
			Hostname:  hostname,
			Time:      stop.Sub(r.beginning),
		},
	}
	r.handler.OnMeasurement(m)
	return addrs, err
}
