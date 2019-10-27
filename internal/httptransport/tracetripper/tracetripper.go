// Package tracetripper contains the tracing round tripper
package tracetripper

import (
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/ooni/netx/internal/connid"
	"github.com/ooni/netx/internal/dialid"
	"github.com/ooni/netx/internal/transactionid"
	"github.com/ooni/netx/model"
)

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
	root := model.ContextMeasurementRootOrDefault(req.Context())

	tid := transactionid.ContextTransactionID(req.Context())
	root.Handler.OnMeasurement(model.Measurement{
		HTTPRoundTripStart: &model.HTTPRoundTripStartEvent{
			DialID:        dialid.ContextDialID(req.Context()),
			Method:        req.Method,
			Time:          time.Now().Sub(root.Beginning),
			TransactionID: tid,
			URL:           req.URL.String(),
		},
	})

	// Prepare a tracer for delivering events
	tracer := &httptrace.ClientTrace{
		TLSHandshakeStart: func() {
			// Event emitted by net/http when DialTLS is not
			// configured in the http.Transport
			root.Handler.OnMeasurement(model.Measurement{
				TLSHandshakeStart: &model.TLSHandshakeStartEvent{
					Time:          time.Now().Sub(root.Beginning),
					TransactionID: tid,
				},
			})
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			// Event emitted by net/http when DialTLS is not
			// configured in the http.Transport
			root.Handler.OnMeasurement(model.Measurement{
				TLSHandshakeDone: &model.TLSHandshakeDoneEvent{
					ConnectionState: model.NewTLSConnectionState(state),
					Error:           err,
					Time:            time.Now().Sub(root.Beginning),
					TransactionID:   tid,
				},
			})
		},
		GotConn: func(info httptrace.GotConnInfo) {
			root.Handler.OnMeasurement(model.Measurement{
				HTTPConnectionReady: &model.HTTPConnectionReadyEvent{
					ConnID: connid.Compute(
						info.Conn.LocalAddr().Network(),
						info.Conn.LocalAddr().String(),
					),
					Network:       info.Conn.LocalAddr().Network(),
					RemoteAddress: info.Conn.RemoteAddr().String(),
					Time:          time.Now().Sub(root.Beginning),
					TransactionID: tid,
				},
			})
		},
		WroteHeaderField: func(key string, values []string) {
			root.Handler.OnMeasurement(model.Measurement{
				HTTPRequestHeader: &model.HTTPRequestHeaderEvent{
					Key:           key,
					Time:          time.Now().Sub(root.Beginning),
					TransactionID: tid,
					Value:         values,
				},
			})
		},
		WroteHeaders: func() {
			root.Handler.OnMeasurement(model.Measurement{
				HTTPRequestHeadersDone: &model.HTTPRequestHeadersDoneEvent{
					Time:          time.Now().Sub(root.Beginning),
					TransactionID: tid,
				},
			})
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			root.Handler.OnMeasurement(model.Measurement{
				HTTPRequestDone: &model.HTTPRequestDoneEvent{
					Error:         info.Err,
					Time:          time.Now().Sub(root.Beginning),
					TransactionID: tid,
				},
			})
		},
		GotFirstResponseByte: func() {
			root.Handler.OnMeasurement(model.Measurement{
				HTTPResponseStart: &model.HTTPResponseStartEvent{
					Time:          time.Now().Sub(root.Beginning),
					TransactionID: tid,
				},
			})
		},
	}

	// If we don't have already a tracer this is a toplevel request, so just
	// set the tracer. Otherwise, we're doing DoH. We cannot set anothert trace
	// because they'd be merged. Instead, replace the existing trace content
	// with the new trace and then remember to reset it.
	origtracer := httptrace.ContextClientTrace(req.Context())
	if origtracer != nil {
		bkp := *origtracer
		*origtracer = *tracer
		defer func() {
			*origtracer = bkp
		}()
	} else {
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), tracer))
	}

	resp, err := t.roundTripper.RoundTrip(req)
	event := &model.HTTPRoundTripDoneEvent{
		Error:         err,
		Time:          time.Now().Sub(root.Beginning),
		TransactionID: tid,
	}
	if resp != nil {
		event.Headers = resp.Header
		event.StatusCode = int64(resp.StatusCode)
	}
	root.Handler.OnMeasurement(model.Measurement{
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
