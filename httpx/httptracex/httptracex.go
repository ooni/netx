// Package httptracex contains OONI's net/http/httptrace extensions.
package httptracex

import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"
)

// EventID is the identifier of an event.
type EventID string

const (
	// HTTPRequestStart is emitted when we're starting the round trip.
	HTTPRequestStart = EventID("HTTPRequestStart")

	// DNSStart is emitted when we start the DNS lookup.
	DNSStart = EventID("DNSStart")

	// DNSDone is emitted when the DNS lookup is complete.
	DNSDone = EventID("DNSDone")

	// ConnectStart is emitted when we start connecting.
	ConnectStart = EventID("ConnectStart")

	// ConnectDone is emitted when we are done connecting.
	ConnectDone = EventID("ConnectDone")

	// TLSHandshakeStart is emitted when the handshake starts.
	TLSHandshakeStart = EventID("TLSHandshakeStart")

	// TLSHandshakeDone is emitted when the handshake is complete.
	TLSHandshakeDone = EventID("TLSHandshakeDone")

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
	// Address is the address used for connecting (e.g. "130.192.91.211:80")
	Address string `json:",omitempty"`

	// Addresses is the list of addresses returned by the DNS
	Addresses []net.IPAddr `json:",omitempty"`

	// Error is the error that occurred
	Error error `json:",omitempty"`

	// EventID is the event identifier
	EventID EventID

	// HeaderKey is a header's key
	HeaderKey string `json:",omitempty"`

	// HeaderValues contains a header's values
	HeaderValues []string `json:",omitempty"`

	// Hostname is the hostname passed to the DNS
	Hostname string `json:",omitempty"`

	// Method is the request method
	Method string `json:",omitempty"`

	// Network is the type of network used for connecting (e.g. "tcp")
	Network string `json:",omitempty"`

	// NumBytes contains the number of transferred bytes
	NumBytes int `json:",omitempty"`

	// StatusCode contains the HTTP status code
	StatusCode int `json:",omitempty"`

	// Time is the time when the event occurred relative to the
	// beginning time stored inside of the EventsContainer
	Time time.Duration

	// TLSConnectionState is the TLS connection state
	TLSConnectionState *tls.ConnectionState `json:",omitempty"`

	// URL is the request URL
	URL string `json:",omitempty"`
}

// EventsContainer contains a summary of round trip events.
type EventsContainer struct {
	// Beginning is when this trace begins
	Beginning time.Time

	// Events contains the events that occurred.
	Events []Event

	mutex sync.Mutex
}

func (ec *EventsContainer) append(ev Event) {
	ec.mutex.Lock()
	ec.Events = append(ec.Events, ev)
	ec.mutex.Unlock()
}

type ctxKey struct{} // same pattern as in net/http/httptrace

func withEventsContainer(ctx context.Context, ec *EventsContainer) context.Context {
	return context.WithValue(httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			ec.append(Event{
				EventID:  DNSStart,
				Hostname: info.Host,
				Time:     time.Now().Sub(ec.Beginning),
			})
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			ec.append(Event{
				Addresses: info.Addrs,
				Error:     info.Err,
				EventID:   DNSDone,
				Time:      time.Now().Sub(ec.Beginning),
			})
		},
		ConnectStart: func(network, addr string) {
			ec.append(Event{
				Address: addr,
				EventID: ConnectStart,
				Network: network,
				Time:    time.Now().Sub(ec.Beginning),
			})
		},
		ConnectDone: func(network, addr string, err error) {
			ec.append(Event{
				Address: addr,
				Error:   err,
				EventID: ConnectDone,
				Network: network,
				Time:    time.Now().Sub(ec.Beginning),
			})
		},
		TLSHandshakeStart: func() {
			ec.append(Event{
				EventID: TLSHandshakeStart,
				Time:    time.Now().Sub(ec.Beginning),
			})
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			ec.append(Event{
				Error:              err,
				EventID:            TLSHandshakeDone,
				TLSConnectionState: &state,
				Time:               time.Now().Sub(ec.Beginning),
			})
		},
		WroteHeaderField: func(key string, values []string) {
			ec.append(Event{
				EventID:      HTTPRequestHeader,
				HeaderKey:    key,
				HeaderValues: values,
				Time:         time.Now().Sub(ec.Beginning),
			})
		},
		WroteHeaders: func() {
			ec.append(Event{
				EventID: HTTPRequestHeadersDone,
				Time:    time.Now().Sub(ec.Beginning),
			})
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			ec.append(Event{
				Error:   info.Err,
				EventID: HTTPRequestDone,
				Time:    time.Now().Sub(ec.Beginning),
			})
		},
		GotFirstResponseByte: func() {
			ec.append(Event{
				EventID: HTTPFirstResponseByte,
				Time:    time.Now().Sub(ec.Beginning),
			})
		},
	}), ctxKey{}, ec)
}

func traceableRequest(req *http.Request, ec *EventsContainer) *http.Request {
	return req.WithContext(withEventsContainer(req.Context(), ec))
}

// Tracer performs an HTTP round trip and records events.
type Tracer struct {
	http.RoundTripper

	// EventsContainer contains events occurred during round trips.
	EventsContainer EventsContainer
}

// RoundTrip peforms the HTTP round trip.
func (rt *Tracer) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	ec := &rt.EventsContainer
	req = traceableRequest(req, ec)
	ec.append(Event{
		EventID: HTTPRequestStart,
		Method:  req.Method,
		Time:    time.Now().Sub(ec.Beginning),
		URL:     req.URL.String(),
	})
	resp, err = rt.RoundTripper.RoundTrip(req) // use child RoundTripper
	if err != nil {
		return
	}
	ec.append(Event{
		EventID:    HTTPResponseStatusCode,
		StatusCode: resp.StatusCode,
		Time:       time.Now().Sub(ec.Beginning),
	})
	for key, values := range resp.Header {
		ec.append(Event{
			EventID:      HTTPResponseHeader,
			HeaderKey:    key,
			HeaderValues: values,
			Time:         time.Now().Sub(ec.Beginning),
		})
	}
	ec.append(Event{
		Error:   err,
		EventID: HTTPResponseHeadersDone,
		Time:    time.Now().Sub(ec.Beginning),
	})
	body := resp.Body
	defer body.Close()
	data, err := ioutil.ReadAll(body)
	if err == nil {
		resp.Body = ioutil.NopCloser(bytes.NewReader(data)) // actionable body
	}
	ec.append(Event{
		Error:    err,
		EventID:  HTTPResponseDone,
		NumBytes: len(data),
		Time:     time.Now().Sub(ec.Beginning),
	})
	return
}
