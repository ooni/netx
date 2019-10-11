// Package tracing contains code for tracing low-level events.
package tracing

import (
	"context"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/model"
)

type contextkey struct{}

// WithHandler returns a copy of ctx with handler set as handler.
func WithHandler(ctx context.Context, handler model.Handler) context.Context {
	if handler == nil {
		panic("nil handler") // like httptrace.WithClientTrace
	}
	return context.WithValue(ctx, contextkey{}, Compose(
		handler, contextHandler(ctx),
	))
}

func contextHandler(ctx context.Context) model.Handler {
	handler, _ := ctx.Value(contextkey{}).(model.Handler)
	return handler
}

// ContextHandler returns the handler set within the context. If no handler
// is set, this function returns the handlers.NoHandler handler.
func ContextHandler(ctx context.Context) model.Handler {
	handler := contextHandler(ctx)
	if handler == nil {
		handler = handlers.NoHandler
	}
	return handler
}

type chain struct {
	handler model.Handler
	next    model.Handler
}

func (c *chain) OnMeasurement(m model.Measurement) {
	c.handler.OnMeasurement(m)
	if c.next != nil {
		c.next.OnMeasurement(m)
	}
}

// Compose returns a single handler that is the composition of
// the two handlers provided as arguments.
func Compose(first, second model.Handler) model.Handler {
	return &chain{
		handler: first,
		next:    second,
	}
}
