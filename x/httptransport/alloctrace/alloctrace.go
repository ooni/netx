// Package alloctrace allocates the client trace.
package alloctrace

import (
	"net/http"
	"net/http/httptrace"
)

// Transport is the transport that allocates the client trace
type Transport struct {
	transport http.RoundTripper
}

// New creates a new Transport
func New(rt http.RoundTripper) *Transport {
	return &Transport{transport: rt}
}

// RoundTrip performs an HTTP round trip.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if tracer := httptrace.ContextClientTrace(ctx); tracer != nil {
		// If we already have a trace, we're recursing. In such case we need to
		// clear the trace, or the output will be completely useless. This is
		// the case when we are using DoH to resolve while inside a round trip.
		saved := new(httptrace.ClientTrace)
		pristine := new(httptrace.ClientTrace)
		*saved = *tracer
		*tracer = *pristine
		defer func() {
			*tracer = *saved
		}()
	} else {
		ctx = httptrace.WithClientTrace(ctx, new(httptrace.ClientTrace))
		req = req.WithContext(ctx)
	}
	return t.transport.RoundTrip(req)
}

// CloseIdleConnections closes the idle connections.
func (t *Transport) CloseIdleConnections() {
	// Adapted from net/http code
	type closeIdler interface {
		CloseIdleConnections()
	}
	if tr, ok := t.transport.(closeIdler); ok {
		tr.CloseIdleConnections()
	}
}
