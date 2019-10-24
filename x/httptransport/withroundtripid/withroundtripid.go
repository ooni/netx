// Package withroundtripid adds the round trip ID
package withroundtripid

import (
	"net/http"

	"github.com/ooni/netx/x/roundtripid"
)

// Transport is the transport that adds the round trip ID
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
	ctx = roundtripid.WithRoundTripID(ctx)
	req = req.WithContext(ctx)
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
