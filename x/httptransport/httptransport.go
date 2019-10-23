// Package httptransport contains the HTTP transport
package httptransport

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/dialer"
	"github.com/ooni/netx/x/resolver"
)

var roundTripID int64

// Transport is the HTTP transport
type Transport struct {
	beginning   time.Time
	handler     model.Handler
	includeBody bool
	transport   http.RoundTripper
}

// New creates a new HTTP transport
func New(
	beginning time.Time,
	handler model.Handler,
	rt http.RoundTripper,
	includeBody bool,
) *Transport {
	return &Transport{
		beginning:   beginning,
		handler:     handler,
		includeBody: includeBody,
		transport:   rt,
	}
}

// RoundTrip performs an HTTP round trip.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	rtID := atomic.AddInt64(&roundTripID, 1)
	ctx := req.Context()
	var mu sync.Mutex
	headers := http.Header{}
	if tracer := httptrace.ContextClientTrace(ctx); tracer != nil {
		panic("tracer already set") // confusing bug I don't wanna see again
	}
	tracer := new(httptrace.ClientTrace)
	ctx = httptrace.WithClientTrace(ctx, tracer)
	tracer.GotConn = func(ci httptrace.GotConnInfo) {
		t.handler.OnMeasurement(model.Measurement{
			HTTPConnectionReady: &model.HTTPConnectionReadyEvent{
				ConnHash:      dialer.ConnHash(ci.Conn),
				DoHDialID:     resolver.ContextDialID(ctx), // or zero
				Network:       ci.Conn.RemoteAddr().Network(),
				RemoteAddress: ci.Conn.RemoteAddr().String(),
				Time:          time.Now().Sub(t.beginning),
				TransactionID: rtID,
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
				TransactionID: rtID,
				URL:           req.URL.String(),
			},
		})
	}
	tracer.WroteRequest = func(info httptrace.WroteRequestInfo) {
		t.handler.OnMeasurement(model.Measurement{
			HTTPRequestDone: &model.HTTPRequestDoneEvent{
				Error:         info.Err,
				Time:          time.Now().Sub(t.beginning),
				TransactionID: rtID,
			},
		})
	}
	tracer.GotFirstResponseByte = func() {
		t.handler.OnMeasurement(model.Measurement{
			HTTPResponseStart: &model.HTTPResponseStartEvent{
				Time:          time.Now().Sub(t.beginning),
				TransactionID: rtID,
			},
		})
	}
	req = req.WithContext(ctx)
	resp, err := t.transport.RoundTrip(req)
	m := model.Measurement{
		HTTPResponseHeadersDone: &model.HTTPResponseHeadersDoneEvent{
			Error:         err,
			Time:          time.Now().Sub(t.beginning),
			TransactionID: rtID,
		},
	}
	if err == nil {
		m.HTTPResponseHeadersDone.StatusCode = int64(resp.StatusCode)
		m.HTTPResponseHeadersDone.Headers = resp.Header
	}
	t.handler.OnMeasurement(m)
	if err == nil && t.includeBody {
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
				TransactionID: rtID,
			},
		})
		t.handler.OnMeasurement(model.Measurement{
			HTTPResponseDone: &model.HTTPResponseDoneEvent{
				Time:          stop.Sub(t.beginning),
				TransactionID: rtID,
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
