// Package tracetripper contains the tracing round tripper
package tracetripper

import (
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/ooni/netx/internal/connid"
	"github.com/ooni/netx/internal/httptransport/transactioner"
	"github.com/ooni/netx/model"
)

// Transport performs single HTTP transactions.
type Transport struct {
	beginning    time.Time
	handler      model.Handler
	roundTripper http.RoundTripper
}

// New creates a new Transport.
func New(
	beginning time.Time, handler model.Handler,
	roundTripper http.RoundTripper,
) *Transport {
	return &Transport{
		beginning:    beginning,
		handler:      handler,
		roundTripper: roundTripper,
	}
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	tid := transactioner.ContextTransactionID(req.Context())
	t.handler.OnMeasurement(model.Measurement{
		HTTPRoundTripStart: &model.HTTPRoundTripStartEvent{
			Method:        req.Method,
			Time:          time.Now().Sub(t.beginning),
			TransactionID: tid,
			URL:           req.URL.String(),
		},
	})
	tracer := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			t.handler.OnMeasurement(model.Measurement{
				HTTPConnectionReady: &model.HTTPConnectionReadyEvent{
					ConnID: connid.Compute(
						info.Conn.LocalAddr().Network(),
						info.Conn.LocalAddr().String(),
					),
					Network:       info.Conn.LocalAddr().Network(),
					RemoteAddress: info.Conn.RemoteAddr().String(),
					Time:          time.Now().Sub(t.beginning),
					TransactionID: tid,
				},
			})
		},
		WroteHeaderField: func(key string, values []string) {
			t.handler.OnMeasurement(model.Measurement{
				HTTPRequestHeader: &model.HTTPRequestHeaderEvent{
					Key:           key,
					Time:          time.Now().Sub(t.beginning),
					TransactionID: tid,
					Value:         values,
				},
			})
		},
		WroteHeaders: func() {
			t.handler.OnMeasurement(model.Measurement{
				HTTPRequestHeadersDone: &model.HTTPRequestHeadersDoneEvent{
					Time:          time.Now().Sub(t.beginning),
					TransactionID: tid,
				},
			})
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			t.handler.OnMeasurement(model.Measurement{
				HTTPRequestDone: &model.HTTPRequestDoneEvent{
					Time:          time.Now().Sub(t.beginning),
					TransactionID: tid,
				},
			})
		},
		GotFirstResponseByte: func() {
			t.handler.OnMeasurement(model.Measurement{
				HTTPResponseStart: &model.HTTPResponseStartEvent{
					Time:          time.Now().Sub(t.beginning),
					TransactionID: tid,
				},
			})
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), tracer))
	resp, err := t.roundTripper.RoundTrip(req)
	event := &model.HTTPRoundTripDoneEvent{
		Error:         err,
		Time:          time.Now().Sub(t.beginning),
		TransactionID: tid,
	}
	if resp != nil {
		event.Headers = resp.Header
		event.StatusCode = int64(resp.StatusCode)
	}
	t.handler.OnMeasurement(model.Measurement{
		HTTPRoundTripDone: event,
	})
	return resp, err
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
