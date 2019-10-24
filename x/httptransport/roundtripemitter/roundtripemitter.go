// Package roundtripemitter emits round trip events
package roundtripemitter

import (
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/dialid"
	"github.com/ooni/netx/x/internal"
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
	if tracer := httptrace.ContextClientTrace(ctx); tracer != nil {
		var mu sync.Mutex
		headers := http.Header{}
		ctx = httptrace.WithClientTrace(ctx, tracer)
		tracer.GotConn = func(ci httptrace.GotConnInfo) {
			t.handler.OnMeasurement(model.Measurement{
				HTTPConnectionReady: &model.HTTPConnectionReadyEvent{
					ConnHash:      internal.ConnHash(ci.Conn),
					DoHDialID:     dialid.ContextDialID(ctx),
					Network:       ci.Conn.RemoteAddr().Network(),
					RemoteAddress: ci.Conn.RemoteAddr().String(),
					Time:          time.Now().Sub(t.beginning),
					TransactionID: roundtripid.ContextRoundTripID(ctx),
				},
			})
		}
		tracer.WroteHeaderField = func(key string, values []string) {
			mu.Lock()
			headers[key] = values
			mu.Unlock()
		}
		tracer.WroteHeaders = func() {
			mu.Lock()
			defer mu.Unlock()
			t.handler.OnMeasurement(model.Measurement{
				HTTPRequestHeadersDone: &model.HTTPRequestHeadersDoneEvent{
					Headers:       headers,
					Method:        req.Method,
					Time:          time.Now().Sub(t.beginning),
					TransactionID: roundtripid.ContextRoundTripID(ctx),
					URL:           req.URL.String(),
				},
			})
		}
		tracer.WroteRequest = func(info httptrace.WroteRequestInfo) {
			t.handler.OnMeasurement(model.Measurement{
				HTTPRequestDone: &model.HTTPRequestDoneEvent{
					Error:         info.Err,
					Time:          time.Now().Sub(t.beginning),
					TransactionID: roundtripid.ContextRoundTripID(ctx),
				},
			})
		}
		tracer.GotFirstResponseByte = func() {
			t.handler.OnMeasurement(model.Measurement{
				HTTPResponseStart: &model.HTTPResponseStartEvent{
					Time:          time.Now().Sub(t.beginning),
					TransactionID: roundtripid.ContextRoundTripID(ctx),
				},
			})
		}
		req = req.WithContext(ctx)
	}
	resp, err := t.transport.RoundTrip(req)
	m := model.Measurement{
		HTTPResponseHeadersDone: &model.HTTPResponseHeadersDoneEvent{
			Error:         err,
			Time:          time.Now().Sub(t.beginning),
			TransactionID: roundtripid.ContextRoundTripID(ctx),
		},
	}
	if err == nil {
		m.HTTPResponseHeadersDone.StatusCode = int64(resp.StatusCode)
		m.HTTPResponseHeadersDone.Headers = resp.Header
	}
	t.handler.OnMeasurement(m)
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
