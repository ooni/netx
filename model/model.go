// Package model contains the data model.
package model

import (
	"context"
	"net"
	"net/http"
	"time"
)

// BaseEvent is the base event.
type BaseEvent struct {
	// ConnID is the ID of the connection we're using or zero
	// if we're not operating on a connection. Note that the
	// connection IDs are never reused.
	ConnID int64 `json:",omitempty"`

	// ElapsedTime is the moment where the event was fired measured
	// as elapsed nanoseconds since a zero moment in time. We use
	// a monotonic clock to compute the ElapsedTime.
	ElapsedTime time.Duration

	// HTTPRoundTripID is the ID of the current HTTP round trip
	// or zero if we're not doing an HTTP round trip. Note that the
	// we will not reuse round trip IDs.
	HTTPRoundTripID int64 `json:",omitempty"`

	// ResolveID is the ID of the resolve currently in progress
	// or zero if we're not currently doing a resolve. Note
	// that we will not reuse resolve IDs.
	ResolveID int64 `json:",omitempty"`
}

// SyscallEvent is an event describing a syscall.
type SyscallEvent struct {
	BaseEvent

	// BlockedTime is the number of nanoseconds we were blocked
	// waiting for the syscall to complete. We use a monotonic
	// block to compute how much time we were blocked.
	BlockedTime time.Duration
}

// ConnectEvent is emitted when connect() returns.
type ConnectEvent struct {
	SyscallEvent
	Error         error
	Network       string
	RemoteAddress string
}

// DNSMessage is a DNS message.
type DNSMessage struct {
	Data []byte
}

// DNSQueryEvent is emitted when we send a DNS query
type DNSQueryEvent struct {
	BaseEvent
	Message DNSMessage
}

// DNSReplyEvent is emitted when we receive a DNS reply
type DNSReplyEvent struct {
	BaseEvent
	Message DNSMessage
}

// HTTPConnectionReadyEvent is emitted when a connection is ready for HTTP.
type HTTPConnectionReadyEvent struct {
	BaseEvent
}

// HTTPRequestStartEvent is emitted when we start sending the request.
type HTTPRequestStartEvent struct {
	BaseEvent
}

// HTTPRequestHeadersDoneEvent is emitted when we have written the headers.
type HTTPRequestHeadersDoneEvent struct {
	BaseEvent
	Headers http.Header
	Method  string
	URL     string
}

// HTTPRequestDoneEvent is emitted when we have sent the body.
type HTTPRequestDoneEvent struct {
	BaseEvent
}

// HTTPResponseStartEvent is emitted when we receive the first response byte.
type HTTPResponseStartEvent struct {
	BaseEvent
}

// HTTPResponseHeadersDoneEvent is emitted after we have received the headers.
type HTTPResponseHeadersDoneEvent struct {
	BaseEvent
	Headers    http.Header
	StatusCode int64
}

// HTTPResponseBodyPartEvent is emitted after we have received
// a part of the response body, or an error reading it. Note that
// bytes read here does not necessarily match bytes returned by
// ReadEvent because of (1) transparent gzip decompression by Go,
// (2) HTTP overhead (headers and chunked body), (3) TLS. This
// is the reason why we also want to record the error here rather
// than just recording the error in ReadEvent.
type HTTPResponseBodyPartEvent struct {
	BaseEvent
	Error    error
	Data     []byte
	NumBytes int64
}

// HTTPResponseDoneEvent is emitted after we have received the body.
type HTTPResponseDoneEvent struct {
	BaseEvent
}

// ReadEvent is emitted when conn.Read returns.
type ReadEvent struct {
	SyscallEvent
	Error    error
	NumBytes int64
}

// ResolveStartEvent is emitted when resolver.LookupHost starts.
type ResolveStartEvent struct {
	BaseEvent
	Hostname string
}

// ResolveDoneEvent is emitted when resolver.LookupHost returns.
type ResolveDoneEvent struct {
	BaseEvent
	Addresses []string
	Error     error
}

// TLSConfig contains TLS configurations.
type TLSConfig struct {
	NextProtos []string
	ServerName string
}

// X509Certificate is an x.509 certificate.
type X509Certificate struct {
	// Data contains the certificate bytes in DER format.
	Data []byte
}

// TLSConnectionState contains the TLS connection state.
type TLSConnectionState struct {
	CipherSuite                uint16
	NegotiatedProtocol         string
	NegotiatedProtocolIsMutual bool
	PeerCertificates           []X509Certificate
	Version                    uint16
}

// TLSHandshakeDoneEvent is emitted when conn.Handshake returns.
type TLSHandshakeDoneEvent struct {
	BaseEvent
	ConnectionState *TLSConnectionState
	Error           error
}

// TLSHandshakeStartEvent is emitted when conn.Handshake starts.
type TLSHandshakeStartEvent struct {
	BaseEvent
	Config TLSConfig
}

// WriteEvent is emitted when conn.Write returns.
type WriteEvent struct {
	SyscallEvent
	Error    error
	NumBytes int64
}

// Measurement contains zero or more events. Do not assume that at any
// time a Measurement will only contain a single event. When a Measurement
// contains an event, the corresponding pointer is non nil.
type Measurement struct {
	Connect                 *ConnectEvent                 `json:",omitempty"`
	DNSQuery                *DNSQueryEvent                `json:",omitempty"`
	DNSReply                *DNSReplyEvent                `json:",omitempty"`
	HTTPConnectionReady     *HTTPConnectionReadyEvent     `json:",omitempty"`
	HTTPRequestStart        *HTTPRequestStartEvent        `json:",omitempty"`
	HTTPRequestHeadersDone  *HTTPRequestHeadersDoneEvent  `json:",omitempty"`
	HTTPRequestDone         *HTTPRequestDoneEvent         `json:",omitempty"`
	HTTPResponseStart       *HTTPResponseStartEvent       `json:",omitempty"`
	HTTPResponseHeadersDone *HTTPResponseHeadersDoneEvent `json:",omitempty"`
	HTTPResponseBodyPart    *HTTPResponseBodyPartEvent    `json:",omitempty"`
	HTTPResponseDone        *HTTPResponseDoneEvent        `json:",omitempty"`
	Read                    *ReadEvent                    `json:",omitempty"`
	ResolveStart            *ResolveStartEvent            `json:",omitempty"`
	ResolveDone             *ResolveDoneEvent             `json:",omitempty"`
	TLSHandshakeStart       *TLSHandshakeStartEvent       `json:",omitempty"`
	TLSHandshakeDone        *TLSHandshakeDoneEvent        `json:",omitempty"`
	Write                   *WriteEvent                   `json:",omitempty"`
}

// Handler handles measurement events.
type Handler interface {
	// OnMeasurement is called when an event occurs. There will be no
	// events after the code that is using the modified Dialer, Transport,
	// or Client is returned. OnMeasurement may be called by background
	// goroutines and OnMeasurement calls may happen concurrently.
	OnMeasurement(Measurement)
}

// DNSClient is a DNS client. The *net.Resolver used by Go implements
// this interface, but other implementations are possible.
//
// This structure is dnsx.Client according to the design document, but
// having it here reduces the import loop headaches. We still export it
// as dnsx.Client inside of the dnsx package.
type DNSClient interface {
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

// DNSRoundTripper represents an abstract DNS transport. Like DNSClient
// this is also available in the dnsx package.
type DNSRoundTripper interface {
	// RoundTrip sends a DNS query and receives the reply.
	RoundTrip(query []byte) (reply []byte, err error)

	// RoundTripContext is like RoundTrip but with a context.
	RoundTripContext(ctx context.Context, query []byte) (reply []byte, err error)
}
