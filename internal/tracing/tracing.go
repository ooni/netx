// Package tracing allows to trace events.
package tracing

import (
	"context"
	"time"

	"github.com/ooni/netx/model"
)

type contextkey struct{}

// Info contains information useful for tracing
type Info struct {
	Beginning     time.Time
	ConnID        int64
	Handler       model.Handler
	TransactionID int64
}

// WithInfo returns a copy of ctx with the specific tracing info
func WithInfo(ctx context.Context, info *Info) context.Context {
	if info == nil {
		panic("nil handler") // like httptrace.WithClientTrace
	}
	return context.WithValue(ctx, contextkey{}, info)
}

// ContextInfo returns the trace info with the context.
func ContextInfo(ctx context.Context) *Info {
	ip, _ := ctx.Value(contextkey{}).(*Info)
	return ip
}
