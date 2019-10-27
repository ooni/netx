// Package model contains the data model. Network events are tagged
// using a unique int64 ConnID. HTTP events also have a unique int64
// ID, TransactionID. Dial events also have their own DialID. A
// zero ID value always means unknown. We never reuse the DialID
// and the TransactionID. For technical reasons, we need to use a
// ConnID that depends on the five tuple, so they're reused.
//
// All events also have a Time. This is always the time in which
// an event has been emitted. We use a monotonic clock. Hence, the
// Time is relative to a predefined zero in time.
//
// Duration, where present, indicates for how long the code
// has been waiting for an event to happen. For example,
// ReadEvent.Duration indicates for how long the code has
// been blocked inside Read().
//
// When an operation may fail, we also include the Error.
package model

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"
)

// CloseEvent is emitted when conn.Close returns.
type CloseEvent struct {
	ConnID   int64
	Duration time.Duration
	Error    error
	Time     time.Duration
}

// ConnectEvent is emitted when connect() returns.
type ConnectEvent struct {
	ConnID        int64
	DialID        int64
	Duration      time.Duration
	Error         error
	Network       string
	RemoteAddress string
	Time          time.Duration
}

// DNSMessage is a DNS message.
type DNSMessage struct {
	Data []byte
}

// DNSQueryEvent is emitted when we send a DNS query
type DNSQueryEvent struct {
	ConnID  int64
	Message DNSMessage
	Time    time.Duration
}

// DNSReplyEvent is emitted when we receive a DNS reply
type DNSReplyEvent struct {
	ConnID  int64
	Message DNSMessage
	Time    time.Duration
}

// HTTPConnectionReadyEvent is emitted when a connection is ready for HTTP.
type HTTPConnectionReadyEvent struct {
	ConnID        int64
	Network       string
	RemoteAddress string
	Time          time.Duration
	TransactionID int64
}

// HTTPRoundTripStartEvent is emitted when we start the round trip.
type HTTPRoundTripStartEvent struct {
	Method        string
	Time          time.Duration
	TransactionID int64
	URL           string
}

// HTTPRequestHeaderEvent is emitted when we have written a header.
type HTTPRequestHeaderEvent struct {
	Key           string
	Time          time.Duration
	TransactionID int64
	Value         []string
}

// HTTPRequestHeadersDoneEvent is emitted when we have written all headers.
type HTTPRequestHeadersDoneEvent struct {
	Time          time.Duration
	TransactionID int64
}

// HTTPRequestDoneEvent is emitted when we have sent the body.
type HTTPRequestDoneEvent struct {
	Error         error
	Time          time.Duration
	TransactionID int64
}

// HTTPResponseStartEvent is emitted when we receive the first response byte.
type HTTPResponseStartEvent struct {
	Time          time.Duration
	TransactionID int64
}

// HTTPRoundTripDoneEvent is emitted at the end of the round trip. Either
// we have an error, or a valid HTTP response.
type HTTPRoundTripDoneEvent struct {
	Error         error
	Headers       http.Header
	StatusCode    int64
	Time          time.Duration
	TransactionID int64
}

// HTTPResponseBodyPartEvent is emitted after we have received
// a part of the response body, or an error reading it. Note that
// bytes read here does not necessarily match bytes returned by
// ReadEvent because of (1) transparent gzip decompression by Go,
// (2) HTTP overhead (headers and chunked body), (3) TLS. This
// is the reason why we also want to record the error here rather
// than just recording the error in ReadEvent.
type HTTPResponseBodyPartEvent struct {
	Error         error
	Data          []byte
	Duration      time.Duration
	NumBytes      int64
	Time          time.Duration
	TransactionID int64
}

// HTTPResponseDoneEvent is emitted after we have received the body.
type HTTPResponseDoneEvent struct {
	Time          time.Duration
	TransactionID int64
}

// ReadEvent is emitted when conn.Read returns.
type ReadEvent struct {
	ConnID   int64
	Duration time.Duration
	Error    error
	NumBytes int64
	Time     time.Duration
}

// ResolveStartEvent is emitted when resolver.LookupHost begins.
type ResolveStartEvent struct {
	DialID   int64
	Hostname string
	Time     time.Duration
}

// ResolveDoneEvent is emitted when resolver.LookupHost returns.
type ResolveDoneEvent struct {
	Addresses []string
	DialID    int64
	Error     error
	Time      time.Duration
}

// X509Certificate is an x.509 certificate.
type X509Certificate struct {
	// Data contains the certificate bytes in DER format.
	Data []byte
}

// TLSConnectionState contains the TLS connection state.
type TLSConnectionState struct {
	CipherSuite        uint16
	NegotiatedProtocol string
	PeerCertificates   []X509Certificate
	Version            uint16
}

// NewTLSConnectionState creates a new TLSConnectionState.
func NewTLSConnectionState(s tls.ConnectionState) TLSConnectionState {
	return TLSConnectionState{
		CipherSuite:        s.CipherSuite,
		NegotiatedProtocol: s.NegotiatedProtocol,
		PeerCertificates:   simplifyCerts(s.PeerCertificates),
		Version:            s.Version,
	}
}

func simplifyCerts(in []*x509.Certificate) (out []X509Certificate) {
	for _, cert := range in {
		out = append(out, X509Certificate{
			Data: cert.Raw,
		})
	}
	return
}

// TLSHandshakeStartEvent is emitted when conn.Handshake starts.
//
// The ConnID field is set when net/http is using DialTLS and hence
// we're calling our TLS dialer. Conversely, the net/http code is
// performing the handshake for us, and TransactionID is set.
type TLSHandshakeStartEvent struct {
	ConnID        int64
	Time          time.Duration
	TransactionID int64
}

// TLSHandshakeDoneEvent is emitted when conn.Handshake returns.
//
// The ConnID field is set when net/http is using DialTLS and hence
// we're calling our TLS dialer. Conversely, the net/http code is
// performing the handshake for us, and TransactionID is set.
type TLSHandshakeDoneEvent struct {
	ConnectionState TLSConnectionState
	ConnID          int64
	Error           error
	Time            time.Duration
	TransactionID   int64
}

// WriteEvent is emitted when conn.Write returns.
type WriteEvent struct {
	ConnID   int64
	Duration time.Duration
	Error    error
	NumBytes int64
	Time     time.Duration
}

// Measurement contains zero or more events. Do not assume that at any
// time a Measurement will only contain a single event. When a Measurement
// contains an event, the corresponding pointer is non nil.
type Measurement struct {
	// DNS
	ResolveStart *ResolveStartEvent `json:",omitempty"`
	ResolveDone  *ResolveDoneEvent  `json:",omitempty"`
	DNSQuery     *DNSQueryEvent     `json:",omitempty"`
	DNSReply     *DNSReplyEvent     `json:",omitempty"`

	// Syscalls
	Connect *ConnectEvent `json:",omitempty"`
	Read    *ReadEvent    `json:",omitempty"`
	Write   *WriteEvent   `json:",omitempty"`
	Close   *CloseEvent   `json:",omitempty"`

	// TLS events
	TLSHandshakeStart *TLSHandshakeStartEvent `json:",omitempty"`
	TLSHandshakeDone  *TLSHandshakeDoneEvent  `json:",omitempty"`

	// HTTP roundtrip events
	HTTPRoundTripStart     *HTTPRoundTripStartEvent     `json:",omitempty"`
	HTTPConnectionReady    *HTTPConnectionReadyEvent    `json:",omitempty"`
	HTTPRequestHeader      *HTTPRequestHeaderEvent      `json:",omitempty"`
	HTTPRequestHeadersDone *HTTPRequestHeadersDoneEvent `json:",omitempty"`
	HTTPRequestDone        *HTTPRequestDoneEvent        `json:",omitempty"`
	HTTPResponseStart      *HTTPResponseStartEvent      `json:",omitempty"`
	HTTPRoundTripDone      *HTTPRoundTripDoneEvent      `json:",omitempty"`

	// HTTP body events
	HTTPResponseBodyPart *HTTPResponseBodyPartEvent `json:",omitempty"`
	HTTPResponseDone     *HTTPResponseDoneEvent     `json:",omitempty"`
}

// Handler handles measurement events.
type Handler interface {
	// OnMeasurement is called when an event occurs. There will be no
	// events after the code that is using the modified Dialer, Transport,
	// or Client is returned. OnMeasurement may be called by background
	// goroutines and OnMeasurement calls may happen concurrently.
	OnMeasurement(Measurement)
}

// DNSResolver is a DNS resolver. The *net.Resolver used by Go implements
// this interface, but other implementations are possible.
type DNSResolver interface {
	// LookupAddr performs a reverse lookup of an address.
	LookupAddr(ctx context.Context, addr string) (names []string, err error)

	// LookupCNAME returns the canonical name of a given host.
	LookupCNAME(ctx context.Context, host string) (cname string, err error)

	// LookupHost resolves a hostname to a list of IP addresses.
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)

	// LookupMX resolves the DNS MX records for a given domain name.
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)

	// LookupNS resolves the DNS NS records for a given domain name.
	LookupNS(ctx context.Context, name string) ([]*net.NS, error)
}

// DNSRoundTripper represents an abstract DNS transport.
type DNSRoundTripper interface {
	// RoundTrip sends a DNS query and receives the reply.
	RoundTrip(ctx context.Context, query []byte) (reply []byte, err error)
}

// Dialer is a dialer for network connections.
type Dialer interface {
	// Dial dials a new connection
	Dial(network, address string) (net.Conn, error)

	// DialContext is like Dial but with context
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// TLSDialer is a dialer for TLS connections.
type TLSDialer interface {
	// DialTLS dials a new TLS connection
	DialTLS(network, address string) (net.Conn, error)

	// DialTLSContext is like DialTLS but with context
	DialTLSContext(ctx context.Context, network, address string) (net.Conn, error)
}

// MeasurementRoot is a measurement root
type MeasurementRoot struct {
	Beginning time.Time
	Handler   Handler
}

type measurementRootContextKey struct{}

type dummyHandler struct{}

func (*dummyHandler) OnMeasurement(Measurement) {}

// ContextMeasurementRoot returns the measurement root configured in the
// provided context, or a nil pointer, if not set.
func ContextMeasurementRoot(ctx context.Context) *MeasurementRoot {
	root, _ := ctx.Value(measurementRootContextKey{}).(*MeasurementRoot)
	return root
}

// ContextMeasurementRootOrDefault returns the measurement root configured in
// the provided context, or a working, dummy, measurement root.
func ContextMeasurementRootOrDefault(ctx context.Context) *MeasurementRoot {
	root := ContextMeasurementRoot(ctx)
	if root == nil {
		root = &MeasurementRoot{
			Beginning: time.Now(),
			Handler:   &dummyHandler{},
		}
	}
	return root
}

// WithMeasurementRoot returns a copy of the context with the
// configured measurement root set. Panics if the provided root
// is a nil pointer, like httptrace.WithClientTrace.
//
// Merging more than one root is not supported. Setting again
// the root is just going to replace the original root.
func WithMeasurementRoot(
	ctx context.Context, root *MeasurementRoot,
) context.Context {
	if root == nil {
		panic("nil measurement root")
	}
	return context.WithValue(
		ctx, measurementRootContextKey{}, root,
	)
}
