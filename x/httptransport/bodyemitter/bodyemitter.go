// Package bodyemitter emits body events
package bodyemitter

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/roundtripid"
)

// Transport is the HTTP transport
type Transport struct {
	beginning time.Time
	handler   model.Handler
	transport http.RoundTripper
}

// New creates a new HTTP transport
func New(
	beginning time.Time,
	handler model.Handler,
	rt http.RoundTripper,
) *Transport {
	return &Transport{
		beginning: beginning,
		handler:   handler,
		transport: rt,
	}
}

// RoundTrip performs an HTTP round trip.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	resp, err := t.transport.RoundTrip(req)
	if err == nil {
		var body []byte // must set outmost `err`
		start := time.Now()
		body, err = ioutil.ReadAll(resp.Body)
		stop := time.Now()
		resp.Body.Close()
		resp.Body = ioutil.NopCloser(bytes.NewReader(body))
		t.handler.OnMeasurement(model.Measurement{
			HTTPResponseBodyPart: &model.HTTPResponseBodyPartEvent{
				Error:         err,
				Data:          body,
				Duration:      stop.Sub(start),
				NumBytes:      int64(len(body)),
				Time:          stop.Sub(t.beginning),
				TransactionID: roundtripid.ContextRoundTripID(ctx),
			},
		})
		t.handler.OnMeasurement(model.Measurement{
			HTTPResponseDone: &model.HTTPResponseDoneEvent{
				Time:          stop.Sub(t.beginning),
				TransactionID: roundtripid.ContextRoundTripID(ctx),
			},
		})
	}
	return resp, err
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
