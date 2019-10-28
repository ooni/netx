// Package model contains the data model.
package model

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

// Measurement contains zero or more events. Do not assume that at any
// time a Measurement will only contain a single event. When a Measurement
// contains an event, the corresponding pointer is non nil.
//
// All events contain a time measurement, `DurationSinceBeginning`, that
// uses a monotonic clock and is relative to a preconfigured "zero".
type Measurement struct {
	// DNS events
	//
	// These are all identifed by a DialID. A ResolveEvent optionally has
	// a reference to the TransactionID that started the dial, if any.
	ResolveStart *ResolveStartEvent `json:",omitempty"`
	DNSQuery     *DNSQueryEvent     `json:",omitempty"`
	DNSReply     *DNSReplyEvent     `json:",omitempty"`
	ResolveDone  *ResolveDoneEvent  `json:",omitempty"`

	// Syscalls
	//
	// These are all identified by a ConnID. A ConnectEvent has a reference
	// to the DialID that caused this connection to be attempted.
	//
	// Because they are syscalls, we don't split them in start/done pairs
	// but we record the amount of time in which we were blocked.
	Connect *ConnectEvent `json:",omitempty"`
	Read    *ReadEvent    `json:",omitempty"`
	Write   *WriteEvent   `json:",omitempty"`
	Close   *CloseEvent   `json:",omitempty"`

	// TLS events
	//
	// Identified by either ConnID or TransactionID. In the former case
	// the TLS handshake is managed by net code, in the latter case it is
	// instead managed by Golang's HTTP engine. It should not happen to
	// have both ConnID and TransactionID different from zero.
	TLSHandshakeStart *TLSHandshakeStartEvent `json:",omitempty"`
	TLSHandshakeDone  *TLSHandshakeDoneEvent  `json:",omitempty"`

	// HTTP roundtrip events
	//
	// A round trip starts when we need a connection to send a request
	// and ends when we've got the response headers or an error.
	//
	// The identifer here is TransactionID, where the transaction is
	// like the round trip except that it terminates when we've finished
	// reading the whole response body.
	HTTPRoundTripStart     *HTTPRoundTripStartEvent     `json:",omitempty"`
	HTTPConnectionReady    *HTTPConnectionReadyEvent    `json:",omitempty"`
	HTTPRequestHeader      *HTTPRequestHeaderEvent      `json:",omitempty"`
	HTTPRequestHeadersDone *HTTPRequestHeadersDoneEvent `json:",omitempty"`
	HTTPRequestDone        *HTTPRequestDoneEvent        `json:",omitempty"`
	HTTPResponseStart      *HTTPResponseStartEvent      `json:",omitempty"`
	HTTPRoundTripDone      *HTTPRoundTripDoneEvent      `json:",omitempty"`

	// HTTP body events
	//
	// They are identified by the TransactionID. You are not going to see
	// these events if you don't fully read response bodies. But that's
	// something you are supposed to do, so you should be fine.
	HTTPResponseBodyPart *HTTPResponseBodyPartEvent `json:",omitempty"`
	HTTPResponseDone     *HTTPResponseDoneEvent     `json:",omitempty"`

	// Extension events.
	//
	// The purpose of these events is to give us some flexibility to
	// experiment with message formats before blessing something as
	// part of the official API of the library. The intent however is
	// to avoid keeping something as an extension for a long time.
	Extension *ExtensionEvent `json:",omitempty"`
}

// ErrWrapper is our error wrapper for Go errors. The key objective of
// this structure is to properly set Failure, which is also returned by
// the Error() method, so be one of the OONI defined strings.
type ErrWrapper struct {
	// ConnID is the connection ID, or zero if not known.
	ConnID int64

	// DialID is the dial ID, or zero if not known.
	DialID int64

	// Failure is the OONI failure string. The failure strings are
	// loosely backward compatible with Measurement Kit.
	//
	// Supported failure strings
	//
	// - `connection_refused`: ECONNREFUSED
	// - `connection_reset`: ECONNRESET
	// - `dns_nxdomain_error`: NXDOMAIN in DNS reply
	// - `eof_error`: unexpected EOF on connection
	// - `generic_timeout_error`: some timer has expired
	// - `ssl_invalid_hostname`: certificate not valid for SNI
	// - `ssl_unknown_autority`: cannot find CA validating certificate
	// - `ssl_invalid_certificate`: e.g. certificate expried
	// - `unknown_failure ...`: any other error
	Failure string

	// TransactionDI is the transaction ID, or zero if not known.
	TransactionID int64

	// WrappedErr is the error that we're wrapping.
	WrappedErr error
}

// Error returns a description of the error that occurred.
func (e *ErrWrapper) Error() string {
	return e.Failure
}

// Unwrap allows to access the underlying error
func (e *ErrWrapper) Unwrap() error {
	return e.WrappedErr
}

// CloseEvent is emitted when the CLOSE syscall returns.
type CloseEvent struct {
	// ConnID is the identifier of this connection.
	ConnID int64

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Error is the error returned by CLOSE.
	Error error

	// SyscallDuration is the number of nanoseconds we were
	// blocked waiting for the syscall to return.
	SyscallDuration time.Duration
}

// ConnectEvent is emitted when the CONNECT syscall returns.
type ConnectEvent struct {
	// ConnID is the identifier of this connection.
	ConnID int64

	// DialID is the identifier of the dial operation as
	// part of which we called CONNECT.
	DialID int64

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Error is the error returned by CONNECT.
	Error error

	// Network is the network we're dialing for, e.g. "tcp"
	Network string

	// RemoteAddress is the remote IP address we're dialing for
	RemoteAddress string

	// SyscallDuration is the number of nanoseconds we were
	// blocked waiting for the syscall to return.
	SyscallDuration time.Duration

	// TransactionID is the ID of the HTTP transaction that caused the
	// current dial to run, or zero if there's no such transaction.
	TransactionID int64 `json:",omitempty"`
}

// DNSQueryEvent is emitted when we send a DNS query.
type DNSQueryEvent struct {
	// Data is the raw data we're sending to the server.
	Data []byte

	// DialID is the identifier of the dial operation as
	// part of which we're sending this query.
	DialID int64

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Msg is the parsed message we're sending to the server.
	Msg *dns.Msg `json:"-"`
}

// DNSReplyEvent is emitted when we receive byte that are
// successfully parsed into a DNS reply.
type DNSReplyEvent struct {
	// Data is the raw data we've received and parsed.
	Data []byte

	// DialID is the identifier of the dial operation as
	// part of which we've received this query.
	DialID int64

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Msg is the received parsed message.
	Msg *dns.Msg `json:"-"`
}

// ExtensionEvent is emitted by a netx extension.
type ExtensionEvent struct {
	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Key is the unique identifier of the event. A good rule of
	// thumb is to use `${packageName}.${messageType}`.
	Key string

	// Severity of the emitted message ("WARN", "INFO", "DEBUG")
	Severity string

	// TransactionID is the identifier of this transaction, provided
	// that we have an active one, otherwise is zero.
	TransactionID int64

	// Value is the extension dependent message. This message
	// has the only requirement of being JSON serializable.
	Value interface{}
}

// HTTPRoundTripStartEvent is emitted when the HTTP transport
// starts the HTTP "round trip". That is, when the transport
// receives from the HTTP client a request to sent. The round
// trip terminates when we receive headers. What we call the
// "transaction" here starts with this event and does not finish
// until we have also finished receiving the response body.
type HTTPRoundTripStartEvent struct {
	// DialID is the identifier of the dial operation that
	// caused this round trip to start. Typically, this occures
	// when doing DoH. If zero, means that this round trip has
	// not been started by any dial operation.
	DialID int64 `json:",omitempty"`

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Method is the request method
	Method string

	// TransactionID is the identifier of this transaction
	TransactionID int64

	// URL is the request URL
	URL string
}

// HTTPConnectionReadyEvent is emitted when the HTTP transport has got
// a connection which is ready for sending the request.
type HTTPConnectionReadyEvent struct {
	// ConnID is the identifier of the connection that is ready. Knowing
	// this ID allows you to bind HTTP events to net events.
	ConnID int64

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// TransactionID is the identifier of this transaction
	TransactionID int64
}

// HTTPRequestHeaderEvent is emitted when we have written a header,
// where written typically means just "buffered".
type HTTPRequestHeaderEvent struct {
	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Key is the header key
	Key string

	// TransactionID is the identifier of this transaction
	TransactionID int64

	// Value is the value/values of this header.
	Value []string
}

// HTTPRequestHeadersDoneEvent is emitted when we have written, or more
// correctly, "buffered" all headers.
type HTTPRequestHeadersDoneEvent struct {
	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// TransactionID is the identifier of this transaction
	TransactionID int64
}

// HTTPRequestDoneEvent is emitted when we have sent the request
// body or there has been any failure in sending the request.
type HTTPRequestDoneEvent struct {
	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Error is non nil if we could not write the request headers or
	// some specific part of the body. When this step of writing
	// the request fails, of course the whole transaction will fail
	// as well. This error however tells you that the issue was
	// when sending the request, not when receiving the response.
	Error error

	// TransactionID is the identifier of this transaction
	TransactionID int64
}

// HTTPResponseStartEvent is emitted when we receive the byte from
// the response on the wire.
type HTTPResponseStartEvent struct {
	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// TransactionID is the identifier of this transaction
	TransactionID int64
}

// HTTPRoundTripDoneEvent is emitted at the end of the round trip. Either
// we have an error, or a valid HTTP response. An error could be caused
// either by not being able to send the request or not being able to receive
// the response. Note that here errors are network/TLS/dialing errors or
// protocol violation errors. No status code will cause errors here.
type HTTPRoundTripDoneEvent struct {
	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Error is the overall result of the round trip. If non-nil, checking
	// also the result of HTTPResponseDone helps to disambiguate whether the
	// error was in sending the request or receiving the response.
	Error error

	// Headers contains the response headers if error is nil.
	Headers http.Header

	// StatusCode contains the HTTP status code if error is nil.
	StatusCode int64

	// TransactionID is the identifier of this transaction
	TransactionID int64
}

// HTTPResponseBodyPartEvent is emitted after we have received
// a part of the response body, or an error reading it. Note that
// bytes read here does not necessarily match bytes returned by
// ReadEvent because of (1) transparent gzip decompression by Go,
// (2) HTTP overhead (headers and chunked body), (3) TLS. This
// is the reason why we also want to record the error here rather
// than just recording the error in ReadEvent.
//
// Note that you are not going to see this event if you do not
// drain the response body, which you're supposed to do, tho.
type HTTPResponseBodyPartEvent struct {
	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Error indicates whether we could not read a part of the body
	Error error

	// Data is a reference to the body we've just read.
	Data []byte

	// TransactionID is the identifier of this transaction
	TransactionID int64
}

// HTTPResponseDoneEvent is emitted after we have received the body,
// when the response body is being closed.
//
// Note that you are not going to see this event if you do not
// drain the response body, which you're supposed to do, tho.
type HTTPResponseDoneEvent struct {
	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// TransactionID is the identifier of this transaction
	TransactionID int64
}

// ReadEvent is emitted when the READ/RECV syscall returns.
type ReadEvent struct {
	// ConnID is the identifier of this connection.
	ConnID int64

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Error is the error returned by READ/RECV.
	Error error

	// NumBytes is the number of bytes received, which may in
	// principle also be nonzero on error.
	NumBytes int64

	// SyscallDuration is the number of nanoseconds we were
	// blocked waiting for the syscall to return.
	SyscallDuration time.Duration
}

// ResolveStartEvent is emitted when we start resolving a domain name.
type ResolveStartEvent struct {
	// DialID is the identifier of the dial operation as
	// part of which we're resolving this domain.
	DialID int64

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Hostname is the domain name to resolve.
	Hostname string

	// TransactionID is the ID of the HTTP transaction that caused the
	// current dial to run, or zero if there's no such transaction.
	TransactionID int64 `json:",omitempty"`
}

// ResolveDoneEvent is emitted when we know the IP addresses of a
// specific domain name, or the resolution failed.
type ResolveDoneEvent struct {
	// Addresses is the list of returned addresses (empty on error).
	Addresses []string

	// DialID is the identifier of the dial operation as
	// part of which we're resolving this domain.
	DialID int64

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Error is the result of the dial operation.
	Error error

	// Hostname is the domain name to resolve.
	Hostname string

	// TransactionID is the ID of the HTTP transaction that caused the
	// current dial to run, or zero if there's no such transaction.
	TransactionID int64 `json:",omitempty"`
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

// TLSHandshakeStartEvent is emitted when the TLS handshake starts.
type TLSHandshakeStartEvent struct {
	// ConnID is the ID of the connection that started the TLS
	// handshake, or zero if we don't know it. Typically, it is
	// zero for connections managed by the HTTP transport, for
	// which we know instead the TransactionID.
	ConnID int64 `json:",omitempty"`

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// TransactionID is the ID of the transaction that started
	// this TLS handshake, or zero if we don't know it. Typically,
	// it is zero for explicit dials, and it's nonzero instead
	// when a connection is managed by HTTP code.
	TransactionID int64 `json:",omitempty"`
}

// TLSHandshakeDoneEvent is emitted when conn.Handshake returns.
type TLSHandshakeDoneEvent struct {
	// ConnectionState is the TLS connection state. Depending on the
	// error type, some fields may have little meaning.
	ConnectionState TLSConnectionState

	// ConnID is the ID of the connection that started the TLS
	// handshake, or zero if we don't know it. Typically, it is
	// zero for connections managed by the HTTP transport, for
	// which we know instead the TransactionID.
	ConnID int64

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Error is the result of the TLS handshake.
	Error error

	// TransactionID is the ID of the transaction that started
	// this TLS handshake, or zero if we don't know it. Typically,
	// it is zero for explicit dials, and it's nonzero instead
	// when a connection is managed by HTTP code.
	TransactionID int64
}

// WriteEvent is emitted when the WRITE/SEND syscall returns.
type WriteEvent struct {
	// ConnID is the identifier of this connection.
	ConnID int64

	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Error is the error returned by WRITE/SEND.
	Error error

	// NumBytes is the number of bytes sent, which may in
	// principle also be nonzero on error.
	NumBytes int64

	// SyscallDuration is the number of nanoseconds we were
	// blocked waiting for the syscall to return.
	SyscallDuration time.Duration
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

// MeasurementRoot is the measurement root.
//
// If you attach this to a context, we'll use it rather than using
// the beginning and hndler configured with resolvers, dialers, HTTP
// clients, and HTTP transports. By attaching a measurement root to
// a context, you can naturally split events by HTTP round trip.
type MeasurementRoot struct {
	// Beginning is the "zero" used to compute the elapsed time.
	Beginning time.Time

	// Handler is the handler that will handle events.
	Handler Handler

	// LookupHost allows to override the host lookup for all the request
	// and dials that use this measurement root.
	LookupHost func(ctx context.Context, hostname string) ([]string, error)
}

type measurementRootContextKey struct{}

type dummyHandler struct{}

func (*dummyHandler) OnMeasurement(Measurement) {}

// ContextMeasurementRoot returns the MeasurementRoot configured in the
// provided context, or a nil pointer, if not set.
func ContextMeasurementRoot(ctx context.Context) *MeasurementRoot {
	root, _ := ctx.Value(measurementRootContextKey{}).(*MeasurementRoot)
	return root
}

// ContextMeasurementRootOrDefault returns the MeasurementRoot configured in
// the provided context, or a working, dummy, MeasurementRoot otherwise.
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
// configured MeasurementRoot set. Panics if the provided root
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
