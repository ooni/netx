// Package bodyreader contains the top HTTP body reader.
package bodyreader

import (
	"io"
	"net/http"
	"time"

	"github.com/ooni/netx/internal/httptransport/transactioner"
	"github.com/ooni/netx/model"
)

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
	tid := transactioner.ContextTransactionID(req.Context())
	resp, err = t.roundTripper.RoundTrip(req)
	if err != nil {
		return
	}
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
