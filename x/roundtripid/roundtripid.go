// Package roundtripid contains code to manage the round trip ID
package roundtripid

import (
	"context"
	"sync/atomic"
)

type contextkey struct{}

var id int64

// WithRoundTripID returns a copy of ctx with a new roundTripID
func WithRoundTripID(ctx context.Context) context.Context {
	return context.WithValue(
		ctx, contextkey{}, atomic.AddInt64(&id, 1),
	)
}

// ContextRoundTripID returns the roundTripID of the context, or zero
func ContextRoundTripID(ctx context.Context) int64 {
	id, _ := ctx.Value(contextkey{}).(int64)
	return id
}
