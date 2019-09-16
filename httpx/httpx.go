// Package httpx contains OONI's net/http extensions.
//
// Client
//
// This package defines an http.Client replacement. Using this
// replacement is definitely the easiest way of using this library to
// perform net-level and http-level measurements.
//
// This is what you need to do:
//
// 1. create a httpx.Client instance with NewClient;
//
// 2. possibly further configure the already configured client.Transport
// and client.Dialer() instances, if needed (for example, you may want
// to configure a logger for both);
//
// 3. pass client.Client to existing code that needs an *http.Client;
//
// 4. when you need it, use client.PopNetMeasurements() to extract
// network level measurements, and use client.PopHTTPMeasurements() to
// extract HTTP level measurements.
//
// Note that step 3 implies that you can use existing Go code
// as long as this code is using a `*http.Client`.
//
// Transport
//
// This package also provides a replacement for http.Transport that during its
// lifecycle will observe and log http-transaction events. Compared to the Client
// replacement, the Transport replacement is more low level.
//
// Each transaction will be identified its unique int64 ID.
//
// The Transport replacement contains a netx.Dialer, which will also perform
// network-level measurements. One of the events logged by the modified
// Transport allow to link network level events to http level events. We never
// reuse IDs, but in theory the int64 counter we use could wrap around.
//
// Measurement is the data structure used by all the events we
// collect. The EventID field of a specific Measurement identifies the
// specific event, and determines what fields are meaningful.
//
// Use transport.PopMeasurements() at any time to extract all the
// measurements collected so far.
//
package httpx

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

	"golang.org/x/net/http2"

	"github.com/bassosimone/netx"
	"github.com/bassosimone/netx/internal"
	"github.com/bassosimone/netx/logx"
)

// EventID is the identifier of an event.
type EventID string

const (
	// GotConnEvent is emitted when we have a connection for this round trip.
	GotConnEvent = EventID("gotConn")

	// RequestStartEvent is emitted when we start sending the request.
	RequestStartEvent = EventID("requestStart")

	// RequestHeadersDoneEvent is emitted when we have written the headers.
	RequestHeadersDoneEvent = EventID("requestHeadersDone")

	// RequestDoneEvent is emitted when we have sent the body.
	RequestDoneEvent = EventID("requestDone")

	// ResponseStartEvent is emitted when we receive the first response byte.
	ResponseStartEvent = EventID("responseStart")

	// ResponseHeadersDoneEvent is emitted after we have received the headers.
	ResponseHeadersDoneEvent = EventID("responseHeadersDone")

	// ResponseDoneEvent is emitted after we have received the body.
	ResponseDoneEvent = EventID("responseDone")
)

// Measurement is an HTTP event measurement. Some fields should be
// always present, others are optional. The optional fields are
// also marked with `json:",omitempty"`.
type Measurement struct {
	// EventID is the event ID.
	EventID EventID

	// ExternalConnID is the identifier of the connection we're using. This allows
	// you to cross link to the the related net events. This field is present
	// when the EventID is GotConnEvent only.
	ExternalConnID string `json:",omitempty"`

	// Headers contains headers for {Request,Response}HeadersDoneEvent.
	Headers []string `json:",omitempty"`

	// Method is the request method for RequestHeadersDoneEvent.
	Method string `json:",omitempty"`

	// StatusCode is the status code for ResponseHeadersDoneEvent.
	StatusCode int `json:",omitempty"`

	// Time is the time when the event occurred relative to the
	// value of Transport.Beginning.
	Time time.Duration

	// TransactionID is the unique identifier of this transaction.
	TransactionID int64

	// URL is the request URL for RequestHeadersDoneEvent.
	URL string `json:",omitempty"`
}

// Transport performs single HTTP transactions and saves MeasurementEvents.
type Transport struct {
	// Beginning is the point in time considered as the beginning of the
	// measurements performed by this Transport. This field is initialized
	// by the NewTransport constructor.
	Beginning time.Time

	// Dialer is the Dialer we'll use. This field is initialized
	// by the NewTransport constructor.
	Dialer *netx.Dialer

	// Logger is the interface used for logging. By default we use a
	// dummy logger that does nothing, but you may want logging.
	Logger logx.Logger

	// Transport is the child Transport. It is initialized
	// by the NewTransport constructor. In particular we will
	// configure it to use Dialer for dialing.
	HTTPTransport *http.Transport

	measurements  []Measurement
	mutex         sync.Mutex
	transactionID int64
}

// NewTransport creates a new Transport.
func NewTransport(beginning time.Time) (transport *Transport, err error) {
	dialer := netx.NewDialer(beginning)
	transport = &Transport{
		Beginning: beginning,
		Dialer:    dialer,
		Logger:    internal.NoLogger{},
		HTTPTransport: &http.Transport{
			Dial:                  dialer.Dial,
			DialContext:           dialer.DialContext,
			DialTLS:               dialer.DialTLS,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          100,
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   10 * time.Second,
		},
	}
	// Configure h2 and make sure that the custom TLSConfig we use for dialing
	// is actually compatible with upgrading to h2. (This mainly means we
	// need to make sure we include "h2" in the NextProtos array.)
	if err = http2.ConfigureTransport(transport.HTTPTransport); err != nil {
		transport = nil
		return
	}
	transport.Dialer.TLSConfig = transport.HTTPTransport.TLSClientConfig.Clone()
	return
}

// PopMeasurements extracts the measurements collected by this Transport
// and returns them in a goroutine safe way.
func (rt *Transport) PopMeasurements() (measurements []Measurement) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()
	measurements = rt.measurements
	rt.measurements = nil
	return
}

func (rt *Transport) appendMeasurement(ev Measurement) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()
	rt.measurements = append(rt.measurements, ev)
}

// roundTripContext is the state private to a specific round trip
type roundTripContext struct {
	incoming      []string   // received headers
	http2         bool       // using http2?
	method        string     // request method
	outgoing      []string   // sent headers
	transactionID int64      // transaction ID
	roundTripper  *Transport // where to save data
	url           *url.URL   // request URL
}

type ctxKey struct{} // same pattern as in net/http/httptrace

func withRoundTripContext(ctx context.Context, rtc *roundTripContext) context.Context {
	return context.WithValue(httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			connid := netx.GetExternalConnID(info.Conn)
			rtc.roundTripper.appendMeasurement(Measurement{
				EventID:        GotConnEvent,
				ExternalConnID: connid, // cross reference
				TransactionID:  rtc.transactionID,
				Time:           time.Now().Sub(rtc.roundTripper.Beginning),
			})
			rtc.roundTripper.Logger.Debugf(
				"(http #%d) got conn <%s>", rtc.transactionID, connid,
			)
		},
		WroteHeaderField: func(key string, values []string) {
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
			rtc.roundTripper.appendMeasurement(Measurement{
				EventID:       RequestHeadersDoneEvent,
				Headers:       rtc.outgoing,
				Method:        rtc.method,
				TransactionID: rtc.transactionID,
				Time:          time.Now().Sub(rtc.roundTripper.Beginning),
				URL:           rtc.url.String(),
			})
			if !rtc.http2 {
				rtc.roundTripper.Logger.Debugf(
					"(http #%d) > %s %s HTTP/1.1", rtc.transactionID,
					rtc.method, rtc.url.RequestURI(),
				)
			}
			for _, s := range rtc.outgoing {
				rtc.roundTripper.Logger.Debugf(
					"(http #%d) > %s", rtc.transactionID, s,
				)
			}
			rtc.roundTripper.Logger.Debugf("(http #%d) >", rtc.transactionID)
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			rtc.roundTripper.appendMeasurement(Measurement{
				EventID:       RequestDoneEvent,
				TransactionID: rtc.transactionID,
				Time:          time.Now().Sub(rtc.roundTripper.Beginning),
			})
			rtc.roundTripper.Logger.Debugf(
				"(http #%d) request sent", rtc.transactionID,
			)
		},
		GotFirstResponseByte: func() {
			rtc.roundTripper.appendMeasurement(Measurement{
				EventID:       ResponseStartEvent,
				TransactionID: rtc.transactionID,
				Time:          time.Now().Sub(rtc.roundTripper.Beginning),
			})
			rtc.roundTripper.Logger.Debugf(
				"(http #%d) start reading response", rtc.transactionID,
			)
		},
	}), ctxKey{}, rtc)
}

func traceableRequest(req *http.Request, rtc *roundTripContext) *http.Request {
	return req.WithContext(withRoundTripContext(req.Context(), rtc))
}

type bodyWrapper struct {
	io.ReadCloser
	roundTripper  *Transport
	transactionID int64
}

func (bw *bodyWrapper) Close() (err error) {
	err = bw.ReadCloser.Close()
	bw.roundTripper.appendMeasurement(Measurement{
		EventID:       ResponseDoneEvent,
		TransactionID: bw.transactionID,
		Time:          time.Now().Sub(bw.roundTripper.Beginning),
	})
	bw.roundTripper.Logger.Debugf("(http #%d) response done", bw.transactionID)
	return
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (rt *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	rtc := &roundTripContext{
		method:        req.Method,
		roundTripper:  rt,
		transactionID: atomic.AddInt64(&rt.transactionID, 1),
		url:           req.URL,
	}
	req = traceableRequest(req, rtc)
	rt.appendMeasurement(Measurement{
		EventID:       RequestStartEvent,
		Method:        req.Method,
		Time:          time.Now().Sub(rt.Beginning),
		TransactionID: rtc.transactionID,
		URL:           req.URL.String(),
	})
	rt.Logger.Debugf(
		"(http #%d) %s %s", rtc.transactionID, req.Method, req.URL.String(),
	)
	resp, err = rt.HTTPTransport.RoundTrip(req)
	if err != nil {
		return
	}
	if rtc.http2 {
		rtc.incoming = append(
			rtc.incoming, fmt.Sprintf(":status: %d", resp.StatusCode),
		)
	}
	for key, values := range resp.Header {
		for _, value := range values {
			rtc.incoming = append(
				rtc.incoming, fmt.Sprintf("%s: %s", key, value),
			)
		}
	}
	rt.appendMeasurement(Measurement{
		EventID:       ResponseHeadersDoneEvent,
		Headers:       rtc.incoming,
		StatusCode:    resp.StatusCode,
		TransactionID: rtc.transactionID,
		Time:          time.Now().Sub(rt.Beginning),
	})
	if rtc.http2 == false {
		rt.Logger.Debugf("(http #%d) < HTTP/%d.%d %d %s", rtc.transactionID,
			resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, resp.Status)
	}
	for _, s := range rtc.incoming {
		rt.Logger.Debugf("(http #%d) < %s", rtc.transactionID, s)
	}
	rt.Logger.Debugf("(http #%d) <", rtc.transactionID)
	// "The http Client and Transport guarantee that Body is always
	//  non-nil, even on responses without a body or responses with
	//  a zero-length body." (from the docs)
	resp.Body = &bodyWrapper{
		ReadCloser:    resp.Body,
		roundTripper:  rt,
		transactionID: rtc.transactionID,
	}
	return
}

// CloseIdleConnections closes any connections which were previously connected
// from previous requests but are now sitting idle in a "keep-alive" state. It
// does not interrupt any connections currently in use.
func (rt *Transport) CloseIdleConnections() {
	rt.HTTPTransport.CloseIdleConnections()
}

// Client is a replacement for http.Client.
type Client struct {
	// HTTPClient is the underlying client. Pass this client to existing code
	// that expects an *http.HTTPClient. Then extract measurements.
	HTTPClient *http.Client

	// Transport is the Transport initially configured to be the RoundTripper
	// used by HTTPClient in the NewClient constructor.
	Transport *Transport
}

// NewClient creates a new client instance.
func NewClient() (*Client, error) {
	transport, err := NewTransport(time.Now())
	if err != nil {
		return nil, err
	}
	return &Client{
		HTTPClient: &http.Client{
			Transport: transport,
		},
		Transport: transport,
	}, nil
}

// Dialer returns the Dialer configured for c.Transport.
func (c *Client) Dialer() *netx.Dialer {
	return c.Transport.Dialer
}

// PopHTTPMeasurements calls c.Transport.PopMeasurements() and returns
// the HTTP level measurements we've performed.
func (c *Client) PopHTTPMeasurements() []Measurement {
	return c.Transport.PopMeasurements()
}

// PopNetMeasurements calls c.Dialer().PopMeasurements() and returns
// the network level measurements we've performed.
func (c *Client) PopNetMeasurements() []netx.Measurement {
	return c.Dialer().PopMeasurements()
}

// SetLogger sets the logger used by c.Dialer() and c.Transport.
func (c *Client) SetLogger(logger logx.Logger) {
	c.Dialer().Logger = logger
	c.Transport.Logger = logger
}

// Beginning returns c.Transport.Beginning, i.e. the time used
// as zero when computing the elapsed time.
func (c *Client) Beginning() time.Time {
	return c.Transport.Beginning
}

// EnableFullTiming configures c.Dialer to record the timing
// of every I/O operation it performs. The default is to exclude
// the operations that are critical to performance.
func (c *Client) EnableFullTiming() {
	c.Dialer().EnableFullTiming = true
}
