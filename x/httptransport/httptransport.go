// Package httptransport contains the HTTP transport
package httptransport

import (
	"net/http"
	"time"

	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/httptransport/alloctrace"
	"github.com/ooni/netx/x/httptransport/bodyemitter"
	"github.com/ooni/netx/x/httptransport/roundtripemitter"
	"github.com/ooni/netx/x/httptransport/withroundtripid"
)

// Transport is the HTTP transport
type Transport struct {
	transport http.RoundTripper
}

// New creates a new HTTP transport
func New(
	beginning time.Time,
	handler model.Handler,
	rt http.RoundTripper,
	includeBody bool,
) *Transport {
	rt = roundtripemitter.New(beginning, handler, rt)
	rt = alloctrace.New(rt)
	if includeBody {
		rt = bodyemitter.New(beginning, handler, rt)
	}
	rt = withroundtripid.New(rt)
	return &Transport{transport: rt}
}

// RoundTrip performs an HTTP round trip.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
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
