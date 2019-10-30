// Package tracetripper contains the tracing round tripper
package tracetripper

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/internal/connid"
	"github.com/ooni/netx/internal/dialid"
	"github.com/ooni/netx/internal/errwrapper"
	"github.com/ooni/netx/internal/transactionid"
	"github.com/ooni/netx/model"
)

// Transport performs single HTTP transactions.
type Transport struct {
	readAllErrs  int64
	readAll      func(r io.Reader) ([]byte, error)
	roundTripper http.RoundTripper
}

// New creates a new Transport.
func New(roundTripper http.RoundTripper) *Transport {
	return &Transport{
		readAll:      ioutil.ReadAll,
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
			DialID:                 dialid.ContextDialID(req.Context()),
			DurationSinceBeginning: time.Now().Sub(root.Beginning),
			Method:                 req.Method,
			TransactionID:          tid,
			URL:                    req.URL.String(),
		},
	})

	var (
		requestHeaders   = http.Header{}
		requestHeadersMu sync.Mutex
	)

	// Prepare a tracer for delivering events
	tracer := &httptrace.ClientTrace{
		TLSHandshakeStart: func() {
			// Event emitted by net/http when DialTLS is not
			// configured in the http.Transport
			root.Handler.OnMeasurement(model.Measurement{
				TLSHandshakeStart: &model.TLSHandshakeStartEvent{
					DurationSinceBeginning: time.Now().Sub(root.Beginning),
					TransactionID:          tid,
				},
			})
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			// Wrapping the error even if we're not returning it because it may
			// less confusing to users to see the wrapped name
			err = errwrapper.SafeErrWrapperBuilder{
				Error:         err,
				TransactionID: tid,
			}.MaybeBuild()
			durationSinceBeginning := time.Now().Sub(root.Beginning)
			root.X.Scoreboard.MaybeTLSHandshakeReset(
				durationSinceBeginning, req.URL, err,
			)
			// Event emitted by net/http when DialTLS is not
			// configured in the http.Transport
			root.Handler.OnMeasurement(model.Measurement{
				TLSHandshakeDone: &model.TLSHandshakeDoneEvent{
					ConnectionState:        model.NewTLSConnectionState(state),
					Error:                  err,
					DurationSinceBeginning: durationSinceBeginning,
					TransactionID:          tid,
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
					DurationSinceBeginning: time.Now().Sub(root.Beginning),
					TransactionID:          tid,
				},
			})
		},
		WroteHeaderField: func(key string, values []string) {
			requestHeadersMu.Lock()
			requestHeaders[key] = values
			requestHeadersMu.Unlock()
			root.Handler.OnMeasurement(model.Measurement{
				HTTPRequestHeader: &model.HTTPRequestHeaderEvent{
					DurationSinceBeginning: time.Now().Sub(root.Beginning),
					Key:                    key,
					TransactionID:          tid,
					Value:                  values,
				},
			})
		},
		WroteHeaders: func() {
			root.Handler.OnMeasurement(model.Measurement{
				HTTPRequestHeadersDone: &model.HTTPRequestHeadersDoneEvent{
					DurationSinceBeginning: time.Now().Sub(root.Beginning),
					TransactionID:          tid,
				},
			})
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			// Wrapping the error even if we're not returning it because it may
			// less confusing to users to see the wrapped name
			err := errwrapper.SafeErrWrapperBuilder{
				Error:         info.Err,
				TransactionID: tid,
			}.MaybeBuild()
			root.Handler.OnMeasurement(model.Measurement{
				HTTPRequestDone: &model.HTTPRequestDoneEvent{
					DurationSinceBeginning: time.Now().Sub(root.Beginning),
					Error:                  err,
					TransactionID:          tid,
				},
			})
		},
		GotFirstResponseByte: func() {
			root.Handler.OnMeasurement(model.Measurement{
				HTTPResponseStart: &model.HTTPResponseStartEvent{
					DurationSinceBeginning: time.Now().Sub(root.Beginning),
					TransactionID:          tid,
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
	err = errwrapper.SafeErrWrapperBuilder{
		Error:         err,
		TransactionID: tid,
	}.MaybeBuild()
	// [*] Require less event joining work by providing info that
	// makes this event alone actionable for OONI
	event := &model.HTTPRoundTripDoneEvent{
		DurationSinceBeginning: time.Now().Sub(root.Beginning),
		Error:                  err,
		RequestHeaders:         requestHeaders,   // [*]
		RequestMethod:          req.Method,       // [*]
		RequestURL:             req.URL.String(), // [*]
		TransactionID:          tid,
	}
	if resp != nil {
		event.Headers = resp.Header
		event.StatusCode = int64(resp.StatusCode)
		// If this is a redirect then Go will ignore the body but we
		// are OONI and we want to see it. Therefore, read it now,
		// dispatch it to the RoundTripDone handler, and make sure we
		// fail the whole round trip if we cannot read it.
		//
		// Also, because redirect responses are supposed to be small,
		// cap their size to 64 KiB, to avoid reading too much.
		//
		// Also, the net/http code really ignores the body but just
		// in case this changes in the future, give it something that
		// implements the same interface as the body.
		if resp.StatusCode >= 301 && resp.StatusCode <= 308 {
			var data []byte
			data, err = t.readAll(io.LimitReader(resp.Body, 1<<17))
			resp.Body.Close()
			event.RedirectBody = data
			resp.Body = ioutil.NopCloser(bytes.NewReader(data))
			if err != nil {
				atomic.AddInt64(&t.readAllErrs, 1)
				resp = nil // this is how net/http likes it
			}
		}
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
