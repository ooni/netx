// Package httptransport contains HTTP transport extensions. Here we
// define a http.Transport that emits events.
package httptransport

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
	"golang.org/x/net/http2"
)

var nextRoundTripID int64

// Transport performs single HTTP transactions and emits
// measurement events as they happen.
type Transport struct {
	http.Transport
}

// NewTransport creates a new Transport.
func NewTransport() *Transport {
	transport := &Transport{
		Transport: http.Transport{
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          100,
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   10 * time.Second,
		},
	}
	// Configure h2 and make sure that the custom TLSConfig we use for dialing
	// is actually compatible with upgrading to h2. (This mainly means we
	// need to make sure we include "h2" in the NextProtos array.) Because
	// http2.ConfigureTransport only returns error when we have already
	// configured http2, it is safe to ignore the return value.
	http2.ConfigureTransport(&transport.Transport)
	return transport
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	outmethod := req.Method
	outurl := req.URL.String()
	tid := atomic.AddInt64(&nextRoundTripID, 1)
	ctx := req.Context()
	tracingInfo := tracing.ContextInfo(ctx)
	if tracingInfo != nil {
		tracingInfo = tracingInfo.CloneWithNewHTTPRoundTripID("httptransport.go", tid)
		req = req.WithContext(tracing.WithInfo(ctx, tracingInfo))
		outheaders := http.Header{}
		var mutex sync.Mutex
		tracer := &httptrace.ClientTrace{
			GotConn: func(info httptrace.GotConnInfo) {
				tracingInfo.Handler.OnMeasurement(model.Measurement{
					HTTPConnectionReady: &model.HTTPConnectionReadyEvent{
						BaseEvent: tracingInfo.BaseEvent(),
					},
				})
			},
			TLSHandshakeStart: func() {
				tracingInfo.EmitTLSHandshakeStart(t.TLSClientConfig)
			},
			TLSHandshakeDone: func(state tls.ConnectionState, err error) {
				tracingInfo.EmitTLSHandshakeDone(&state, err)
			},
			WroteHeaderField: func(key string, values []string) {
				mutex.Lock()
				outheaders[key] = values
				mutex.Unlock()
			},
			WroteHeaders: func() {
				mutex.Lock()
				m := model.Measurement{
					HTTPRequestHeadersDone: &model.HTTPRequestHeadersDoneEvent{
						BaseEvent: tracingInfo.BaseEvent(),
						Headers:   outheaders,
						Method:    outmethod,
						URL:       outurl,
					},
				}
				mutex.Unlock()
				tracingInfo.Handler.OnMeasurement(m)
			},
			WroteRequest: func(info httptrace.WroteRequestInfo) {
				tracingInfo.Handler.OnMeasurement(model.Measurement{
					HTTPRequestDone: &model.HTTPRequestDoneEvent{
						BaseEvent: tracingInfo.BaseEvent(),
					},
				})
			},
			GotFirstResponseByte: func() {
				tracingInfo.Handler.OnMeasurement(model.Measurement{
					HTTPResponseStart: &model.HTTPResponseStartEvent{
						BaseEvent: tracingInfo.BaseEvent(),
					},
				})
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), tracer))
	}
	resp, err = t.Transport.RoundTrip(req)
	if err != nil {
		return
	}
	if tracingInfo != nil {
		tracingInfo.Handler.OnMeasurement(model.Measurement{
			HTTPResponseHeadersDone: &model.HTTPResponseHeadersDoneEvent{
				BaseEvent:  tracingInfo.BaseEvent(),
				Headers:    resp.Header,
				StatusCode: int64(resp.StatusCode),
			},
		})
		// "The http Client and Transport guarantee that Body is always
		//  non-nil, even on responses without a body or responses with
		//  a zero-length body." (from the docs)
		resp.Body = &bodyWrapper{
			ReadCloser:  resp.Body,
			tracingInfo: tracingInfo,
		}
	}
	return
}

type bodyWrapper struct {
	io.ReadCloser
	tracingInfo *tracing.Info
}

func (bw *bodyWrapper) Read(b []byte) (n int, err error) {
	n, err = bw.ReadCloser.Read(b)
	bw.tracingInfo.Handler.OnMeasurement(model.Measurement{
		HTTPResponseBodyPart: &model.HTTPResponseBodyPartEvent{
			BaseEvent: bw.tracingInfo.BaseEvent(),
			// "Read reads up to len(p) bytes into p. It returns the number of
			// bytes read (0 <= n <= len(p)) and any error encountered."
			Data:     b[:n],
			Error:    err,
			NumBytes: int64(n),
		},
	})
	return
}

func (bw *bodyWrapper) Close() (err error) {
	err = bw.ReadCloser.Close()
	bw.tracingInfo.Handler.OnMeasurement(model.Measurement{
		HTTPResponseDone: &model.HTTPResponseDoneEvent{
			BaseEvent: bw.tracingInfo.BaseEvent(),
		},
	})
	return
}
