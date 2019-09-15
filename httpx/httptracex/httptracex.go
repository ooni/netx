// Package httptracex contains OONI's net/http/httptrace extensions.
package httptracex

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bassosimone/netx"
	"github.com/bassosimone/netx/logx"
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
	Logger logx.Logger

	mutex     sync.Mutex
	requestID int64
}

func (ec *EventsContainer) append(ev Event) {
	ec.mutex.Lock()
	ec.Events = append(ec.Events, ev)
	ec.mutex.Unlock()
}

// roundTripContext is the state private to a specific round trip
type roundTripContext struct {
	container *EventsContainer // where to save data
	incoming  []string         // received headers
	http2     bool             // using http2?
	method    string           // request method
	outgoing  []string         // sent headers
	requestID int64            // request ID
	url       *url.URL         // request URL
}

type ctxKey struct{} // same pattern as in net/http/httptrace

func withEventsContainer(
	ctx context.Context, rtc *roundTripContext,
) context.Context {
	ec, id := rtc.container, rtc.requestID
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
			if key == ":method" {
				rtc.http2 = true
			}
			for _, value := range values {
				rtc.outgoing = append(
					rtc.outgoing, fmt.Sprintf("%s: %s", key, value),
				)
			}
		},
		WroteHeaders: func() {
			ec.append(Event{
				EventID:   HTTPRequestHeadersDone,
				RequestID: id,
				Time:      time.Now().Sub(ec.Beginning),
			})
			if !rtc.http2 {
				ec.Logger.Debugf("(http #%d) > %s %s HTTP/1.1", id, rtc.method,
					rtc.url.RequestURI())
			}
			for _, s := range rtc.outgoing {
				ec.Logger.Debugf("(http #%d) > %s", id, s)
			}
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

func traceableRequest(req *http.Request, rtc *roundTripContext) *http.Request {
	return req.WithContext(withEventsContainer(req.Context(), rtc))
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
	rtc := &roundTripContext{
		container: ec,
		method:    req.Method,
		url:       req.URL,
		requestID: reqid,
	}
	req = traceableRequest(req, rtc)
	ec.append(Event{
		EventID:   HTTPRequestStart,
		Method:    req.Method,
		Time:      time.Now().Sub(ec.Beginning),
		RequestID: reqid,
		URL:       req.URL.String(),
	})
	ec.Logger.Debugf("(http #%d) %s %s", reqid, req.Method, req.URL.String())
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
	for key, values := range resp.Header {
		ec.append(Event{
			EventID:      HTTPResponseHeader,
			HeaderKey:    key,
			HeaderValues: values,
			RequestID:    reqid,
			Time:         time.Now().Sub(ec.Beginning),
		})
		for _, value := range values {
			rtc.incoming = append(
				rtc.incoming, fmt.Sprintf("%s: %s", key, value),
			)
		}
	}
	ec.append(Event{
		Error:     err,
		EventID:   HTTPResponseHeadersDone,
		RequestID: reqid,
		Time:      time.Now().Sub(ec.Beginning),
	})
	if rtc.http2 == false {
		ec.Logger.Debugf("(http #%d) < HTTP/%d.%d %d %s", reqid,
			resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, resp.Status)
	}
	for _, s := range rtc.incoming {
		ec.Logger.Debugf("(http #%d) < %s", reqid, s)
	}
	ec.Logger.Debugf("(http #%d) <", reqid)
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
