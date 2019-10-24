// Package tracetripper contains the tracing round tripper
package tracetripper

import (
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

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
	outmethod := req.Method
	outurl := req.URL.String()
	tid := transactioner.ContextTransactionID(req.Context())
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
	return t.roundTripper.RoundTrip(req)
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
