// Package httptracex contains code to help with HTTP tracing
package httptracex

import (
	"context"
	"net/http"

	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
)

// ContextWithHandler returns a copy of ctx with handler set as handler.
func ContextWithHandler(ctx context.Context, handler model.Handler) context.Context {
	return tracing.WithHandler(ctx, handler)
}

// RequestWithHandler returns a copy of req with handler set as handler.
func RequestWithHandler(req *http.Request, handler model.Handler) *http.Request {
	return req.WithContext(ContextWithHandler(req.Context(), handler))
}
