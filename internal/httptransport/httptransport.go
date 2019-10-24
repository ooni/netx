// Package httptransport contains HTTP transport extensions. Here we
// define a http.Transport that emits events.
package httptransport

import (
	"net/http"
	"time"

	"github.com/ooni/netx/internal/httptransport/bodyreader"
	"github.com/ooni/netx/internal/httptransport/tracetripper"
	"github.com/ooni/netx/internal/httptransport/transactioner"
	"github.com/ooni/netx/model"
)

// Transport performs single HTTP transactions and emits
// measurement events as they happen.
type Transport struct {
	roundTripper http.RoundTripper
}

// New creates a new Transport.
func New(
	beginning time.Time, handler model.Handler,
	roundTripper http.RoundTripper,
) *Transport {
	return &Transport{
		roundTripper: transactioner.New(bodyreader.New(
			beginning, handler, tracetripper.New(
				beginning, handler, roundTripper,
			),
		))}
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return t.roundTripper.RoundTrip(req)
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
