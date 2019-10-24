// Package httptransport contains HTTP transport extensions. Here we
// define a http.Transport that emits events.
package httptransport

import (
	"net/http"
	"time"

	"github.com/ooni/netx/internal/httptransport/toptripper"
	"github.com/ooni/netx/internal/httptransport/transactioner"
	"github.com/ooni/netx/model"
	"golang.org/x/net/http2"
)

var nextTransactionID int64

// Transport performs single HTTP transactions and emits
// measurement events as they happen.
type Transport struct {
	Transport    *http.Transport
	Handler      model.Handler
	Beginning    time.Time
	roundTripper http.RoundTripper
}

// NewTransport creates a new Transport.
func NewTransport(beginning time.Time, handler model.Handler) *Transport {
	transport := &Transport{
		Beginning: beginning,
		Handler:   handler,
		Transport: &http.Transport{
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          100,
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   10 * time.Second,
		},
	}
	transport.roundTripper = transactioner.New(toptripper.New(
		beginning, handler, transport.Transport,
	))
	// Configure h2 and make sure that the custom TLSConfig we use for dialing
	// is actually compatible with upgrading to h2. (This mainly means we
	// need to make sure we include "h2" in the NextProtos array.) Because
	// http2.ConfigureTransport only returns error when we have already
	// configured http2, it is safe to ignore the return value.
	http2.ConfigureTransport(transport.Transport)
	return transport
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
