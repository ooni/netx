// Package transactioner contains the transaction assigning round tripper
package transactioner

import (
	"context"
	"net/http"
	"sync/atomic"
)

type contextkey struct{}

var id int64

// WithTransactionID returns a copy of ctx with TransactionID
func WithTransactionID(ctx context.Context) context.Context {
	return context.WithValue(
		ctx, contextkey{}, atomic.AddInt64(&id, 1),
	)
}

// ContextTransactionID returns the TransactionID of the context, or zero
func ContextTransactionID(ctx context.Context) int64 {
	id, _ := ctx.Value(contextkey{}).(int64)
	return id
}

// Transport performs single HTTP transactions.
type Transport struct {
	roundTripper http.RoundTripper
}

// New creates a new Transport.
func New(roundTripper http.RoundTripper) *Transport {
	return &Transport{
		roundTripper: roundTripper,
	}
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := WithTransactionID(req.Context())
	return t.roundTripper.RoundTrip(req.WithContext(ctx))
}

// CloseIdleConnections closes the idle connections.
func (t *Transport) CloseIdleConnections() {
	// Adapted from net/http code
	type closeIdler interface {
		CloseIdleConnections()
	}
	if tr, ok := t.roundTripper.(closeIdler); ok {
		tr.CloseIdleConnections()
	}
}
