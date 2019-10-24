// Package toptripper contains the top HTTP round tripper.
package toptripper

import (
	"io"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/model"
)

var nextTransactionID int64

// Transport performs single HTTP transactions and emits
// measurement events as they happen.
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
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	outmethod := req.Method
	outurl := req.URL.String()
	tid := atomic.AddInt64(&nextTransactionID, 1)
	outheaders := http.Header{}
	var mutex sync.Mutex
	tracer := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			t.handler.OnMeasurement(model.Measurement{
				HTTPConnectionReady: &model.HTTPConnectionReadyEvent{
					LocalAddress:  info.Conn.LocalAddr().String(),
					Network:       info.Conn.LocalAddr().Network(),
					RemoteAddress: info.Conn.RemoteAddr().String(),
					Time:          time.Now().Sub(t.beginning),
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
					Time:          time.Now().Sub(t.beginning),
					TransactionID: tid,
					URL:           outurl,
				},
			}
			mutex.Unlock()
			t.handler.OnMeasurement(m)
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
	resp, err = t.roundTripper.RoundTrip(req)
	if err != nil {
		return
	}
	t.handler.OnMeasurement(model.Measurement{
		HTTPResponseHeadersDone: &model.HTTPResponseHeadersDoneEvent{
			Headers:       resp.Header,
			StatusCode:    int64(resp.StatusCode),
			Time:          time.Now().Sub(t.beginning),
			TransactionID: tid,
		},
	})
	// "The http Client and Transport guarantee that Body is always
	//  non-nil, even on responses without a body or responses with
	//  a zero-length body." (from the docs)
	resp.Body = &bodyWrapper{
		ReadCloser: resp.Body,
		t:          t,
		tid:        tid,
	}
	return
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

type bodyWrapper struct {
	io.ReadCloser
	t   *Transport
	tid int64
}

func (bw *bodyWrapper) Read(b []byte) (n int, err error) {
	start := time.Now()
	n, err = bw.ReadCloser.Read(b)
	stop := time.Now()
	bw.t.handler.OnMeasurement(model.Measurement{
		HTTPResponseBodyPart: &model.HTTPResponseBodyPartEvent{
			// "Read reads up to len(p) bytes into p. It returns the number of
			// bytes read (0 <= n <= len(p)) and any error encountered."
			Data:          b[:n],
			Duration:      stop.Sub(start),
			Error:         err,
			NumBytes:      int64(n),
			Time:          stop.Sub(bw.t.beginning),
			TransactionID: bw.tid,
		},
	})
	return
}

func (bw *bodyWrapper) Close() (err error) {
	err = bw.ReadCloser.Close()
	bw.t.handler.OnMeasurement(model.Measurement{
		HTTPResponseDone: &model.HTTPResponseDoneEvent{
			Time:          time.Now().Sub(bw.t.beginning),
			TransactionID: bw.tid,
		},
	})
	return
}
