// Package httptransport contains HTTP transport extensions. Here we
// define a http.Transport that emits events.
package httptransport

import (
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

var nextTransactionID int64

// Transport performs single HTTP transactions and emits
// measurement events as they happen.
type Transport struct {
	http.Transport
	Handler   model.Handler
	Beginning time.Time
}

// NewTransport creates a new Transport.
func NewTransport(beginning time.Time, handler model.Handler) *Transport {
	transport := &Transport{
		Beginning: beginning,
		Handler:   handler,
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
	handler := tracing.Compose(
		t.Handler, tracing.ContextHandler(req.Context()),
	)
	outmethod := req.Method
	outurl := req.URL.String()
	tid := atomic.AddInt64(&nextTransactionID, 1)
	outheaders := http.Header{}
	var mutex sync.Mutex
	tracer := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			handler.OnMeasurement(model.Measurement{
				HTTPConnectionReady: &model.HTTPConnectionReadyEvent{
					LocalAddress:  info.Conn.LocalAddr().String(),
					Network:       info.Conn.LocalAddr().Network(),
					RemoteAddress: info.Conn.RemoteAddr().String(),
					Time:          time.Now().Sub(t.Beginning),
					TransactionID: tid,
				},
			})
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
					Headers:       outheaders,
					Method:        outmethod,
					Time:          time.Now().Sub(t.Beginning),
					TransactionID: tid,
					URL:           outurl,
				},
			}
			mutex.Unlock()
			handler.OnMeasurement(m)
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			handler.OnMeasurement(model.Measurement{
				HTTPRequestDone: &model.HTTPRequestDoneEvent{
					Time:          time.Now().Sub(t.Beginning),
					TransactionID: tid,
				},
			})
		},
		GotFirstResponseByte: func() {
			handler.OnMeasurement(model.Measurement{
				HTTPResponseStart: &model.HTTPResponseStartEvent{
					Time:          time.Now().Sub(t.Beginning),
					TransactionID: tid,
				},
			})
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), tracer))
	resp, err = t.Transport.RoundTrip(req)
	if err != nil {
		return
	}
	handler.OnMeasurement(model.Measurement{
		HTTPResponseHeadersDone: &model.HTTPResponseHeadersDoneEvent{
			Headers:       resp.Header,
			StatusCode:    int64(resp.StatusCode),
			Time:          time.Now().Sub(t.Beginning),
			TransactionID: tid,
		},
	})
	// "The http Client and Transport guarantee that Body is always
	//  non-nil, even on responses without a body or responses with
	//  a zero-length body." (from the docs)
	resp.Body = &bodyWrapper{
		ReadCloser: resp.Body,
		handler:    handler,
		t:          t,
		tid:        tid,
	}
	return
}

type bodyWrapper struct {
	io.ReadCloser
	handler model.Handler
	t       *Transport
	tid     int64
}

func (bw *bodyWrapper) Read(b []byte) (n int, err error) {
	start := time.Now()
	n, err = bw.ReadCloser.Read(b)
	stop := time.Now()
	bw.handler.OnMeasurement(model.Measurement{
		HTTPResponseBodyPart: &model.HTTPResponseBodyPartEvent{
			// "Read reads up to len(p) bytes into p. It returns the number of
			// bytes read (0 <= n <= len(p)) and any error encountered."
			Data:          b[:n],
			Duration:      stop.Sub(start),
			Error:         err,
			NumBytes:      int64(n),
			Time:          stop.Sub(bw.t.Beginning),
			TransactionID: bw.tid,
		},
	})
	return
}

func (bw *bodyWrapper) Close() (err error) {
	err = bw.ReadCloser.Close()
	bw.handler.OnMeasurement(model.Measurement{
		HTTPResponseDone: &model.HTTPResponseDoneEvent{
			Time:          time.Now().Sub(bw.t.Beginning),
			TransactionID: bw.tid,
		},
	})
	return
}
