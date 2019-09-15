// Package httptracex contains OONI's net/http/httptrace extensions.
package httptracex

import (
	"context"
	"io"
	"net/http"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bassosimone/netx/log"
	"github.com/bassosimone/netx"
)

// EventID is the identifier of an event.
type EventID string

const (
	// GotConn is emitted when we know the connection we'll use.
	GotConn = EventID("GotConn")

	// HTTPRequestStart is emitted when we're starting the round trip.
	HTTPRequestStart = EventID("HTTPRequestStart")

	// HTTPRequestHeader is emitted when we write an HTTP header.
	HTTPRequestHeader = EventID("HTTPRequestHeader")

	// HTTPRequestHeadersDone is emitted when we've written the headers.
	HTTPRequestHeadersDone = EventID("HTTPRequestHeadersDone")

	// HTTPRequestDone is emitted when we're done writing the request.
	HTTPRequestDone = EventID("HTTPRequestDone")

	// HTTPFirstResponseByte is emitted when we receive the first response byte.
	HTTPFirstResponseByte = EventID("HTTPFirstResponseByte")

	// HTTPResponseStatusCode is emitted when we know the status code.
	HTTPResponseStatusCode = EventID("HTTPResponseStatusCode")

	// HTTPResponseHeader is emitted when we know the header.
	HTTPResponseHeader = EventID("HTTPResponseHeader")

	// HTTPResponseHeadersDone is emitted after we've received the headers.
	HTTPResponseHeadersDone = EventID("HTTPResponseHeadersDone")

	// HTTPResponseDone is emitted after we've received the body.
	HTTPResponseDone = EventID("HTTPResponseDone")
)

// Event contains information about an event.
type Event struct {
	// ConnID is the identifier of the connection we'll use for this request.
	ConnID int64 `json:",omitempty"`

	// Error is the error that occurred
	Error error `json:",omitempty"`

	// EventID is the event identifier
	EventID EventID

	// HeaderKey is a header's key
	HeaderKey string `json:",omitempty"`

	// HeaderValues contains a header's values
	HeaderValues []string `json:",omitempty"`

	// Method is the request method
	Method string `json:",omitempty"`

	// NumBytes contains the number of transferred bytes
	NumBytes int `json:",omitempty"`

	// RequestID is the request ID
	RequestID int64

	// StatusCode contains the HTTP status code
	StatusCode int `json:",omitempty"`

	// Time is the time when the event occurred relative to the
	// beginning time stored inside of the EventsContainer
	Time time.Duration

	// URL is the request URL
	URL string `json:",omitempty"`
}

// EventsContainer contains a summary of round trip events.
type EventsContainer struct {
	// Beginning is when this trace begins
	Beginning time.Time

	// Events contains the events that occurred.
	Events []Event

	// Logger is the logger to use.
	Logger log.Logger

	mutex     sync.Mutex
	requestID int64
}

func (ec *EventsContainer) append(ev Event) {
	ec.mutex.Lock()
	ec.Events = append(ec.Events, ev)
	ec.mutex.Unlock()
}

type ctxKey struct{} // same pattern as in net/http/httptrace

func withEventsContainer(
	ctx context.Context, ec *EventsContainer, id int64,
) context.Context {
	return context.WithValue(httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			var connid int64
			if netx.GetConnID(info.Conn, &connid) == false {
				return
			}
			ec.append(Event{
				ConnID:    connid,
				EventID:   GotConn,
				RequestID: id,
				Time:      time.Now().Sub(ec.Beginning),
			})
			ec.Logger.Debugf("(http #%d) got conn #%d", id, connid)
		},
		WroteHeaderField: func(key string, values []string) {
			ec.append(Event{
				EventID:      HTTPRequestHeader,
				HeaderKey:    key,
				HeaderValues: values,
				RequestID:    id,
				Time:         time.Now().Sub(ec.Beginning),
			})
			for _, value := range values {
				ec.Logger.Debugf("(http #%d) > %s: %s", id, key, value)
			}
		},
		WroteHeaders: func() {
			ec.append(Event{
				EventID:   HTTPRequestHeadersDone,
				RequestID: id,
				Time:      time.Now().Sub(ec.Beginning),
			})
			ec.Logger.Debugf("(http #%d) >", id)
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			ec.append(Event{
				Error:     info.Err,
				EventID:   HTTPRequestDone,
				RequestID: id,
				Time:      time.Now().Sub(ec.Beginning),
			})
			ec.Logger.Debugf("(http #%d) request sent", id)
		},
		GotFirstResponseByte: func() {
			ec.append(Event{
				EventID:   HTTPFirstResponseByte,
				RequestID: id,
				Time:      time.Now().Sub(ec.Beginning),
			})
			ec.Logger.Debugf("(http #%d) got first response byte", id)
		},
	}), ctxKey{}, ec)
}

func traceableRequest(req *http.Request, ec *EventsContainer, id int64) *http.Request {
	return req.WithContext(withEventsContainer(req.Context(), ec, id))
}

// Tracer performs an HTTP round trip and records events.
type Tracer struct {
	http.RoundTripper

	// EventsContainer contains events occurred during round trips.
	EventsContainer EventsContainer
}

type bodyWrapper struct {
	io.ReadCloser
	ec    *EventsContainer
	reqid int64
}

func (bw *bodyWrapper) Close() (err error) {
	err = bw.ReadCloser.Close()
	bw.ec.append(Event{
		Error:     err,
		EventID:   HTTPResponseDone,
		RequestID: bw.reqid,
		Time:      time.Now().Sub(bw.ec.Beginning),
	})
	bw.ec.Logger.Debugf("(http #%d) response done", bw.reqid)
	return
}

// RoundTrip peforms the HTTP round trip.
func (rt *Tracer) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	ec := &rt.EventsContainer
	reqid := atomic.AddInt64(&ec.requestID, 1)
	req = traceableRequest(req, ec, reqid)
	ec.append(Event{
		EventID:   HTTPRequestStart,
		Method:    req.Method,
		Time:      time.Now().Sub(ec.Beginning),
		RequestID: reqid,
		URL:       req.URL.String(),
	})
	ec.Logger.Debugf("(http #%d) > %s %s HTTP/...", reqid, req.Method, req.URL.RequestURI())
	resp, err = rt.RoundTripper.RoundTrip(req) // use child RoundTripper
	if err != nil {
		return
	}
	ec.append(Event{
		EventID:    HTTPResponseStatusCode,
		RequestID:  reqid,
		StatusCode: resp.StatusCode,
		Time:       time.Now().Sub(ec.Beginning),
	})
	ec.Logger.Debugf("(http #%d) < HTTP/... %d ...", reqid, resp.StatusCode)
	for key, values := range resp.Header {
		ec.append(Event{
			EventID:      HTTPResponseHeader,
			HeaderKey:    key,
			HeaderValues: values,
			RequestID:    reqid,
			Time:         time.Now().Sub(ec.Beginning),
		})
		for _, value := range values {
			ec.Logger.Debugf("(http #%d) < %s: %s", reqid, key, value)
		}
	}
	ec.Logger.Debugf("(http #%d) <", reqid)
	ec.append(Event{
		Error:     err,
		EventID:   HTTPResponseHeadersDone,
		RequestID: reqid,
		Time:      time.Now().Sub(ec.Beginning),
	})
	// "The http Client and Transport guarantee that Body is always
	//  non-nil, even on responses without a body or responses with
	//  a zero-length body." (from the docs)
	resp.Body = &bodyWrapper{
		ReadCloser: resp.Body,
		ec:         ec,
		reqid:      reqid,
	}
	return
}
