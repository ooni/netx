// Package model contains the data model. Network events are tagged
// using a unique int64 ConnID. HTTP events also have a unique int64
// ID, TransactionID. These IDs are never reused.
//
// To join network events and HTTP events, use the LocalAddress and
// RemoteAddress that are included both in the ConnectEvent and in
// the HTTPConnectionReadyEvent.
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
	Duration      time.Duration
	Error         error
	LocalAddress  string
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
	LocalAddress  string
	Network       string
	RemoteAddress string
	Time          time.Duration
	TransactionID int64
}

// HTTPRequestStartEvent is emitted when we start sending the request.
type HTTPRequestStartEvent struct {
	Time          time.Duration
	TransactionID int64
}

// HTTPRequestHeadersDoneEvent is emitted when we have written the headers.
type HTTPRequestHeadersDoneEvent struct {
	Headers       http.Header
	Method        string
	Time          time.Duration
	TransactionID int64
	URL           string
}

// HTTPRequestDoneEvent is emitted when we have sent the body.
type HTTPRequestDoneEvent struct {
	Time          time.Duration
	TransactionID int64
}

// HTTPResponseStartEvent is emitted when we receive the first response byte.
type HTTPResponseStartEvent struct {
	Time          time.Duration
	TransactionID int64
}

// HTTPResponseHeadersDoneEvent is emitted after we have received the headers.
type HTTPResponseHeadersDoneEvent struct {
	Headers       http.Header
	StatusCode    int64
	Time          time.Duration
	TransactionID int64
}

// HTTPResponseBodyPartEvent is emitted after we have received
// a part of the response body, or an error reading it.
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

// ResolveEvent is emitted when resolver.LookupHost returns.
type ResolveEvent struct {
	Addresses []string
	ConnID    int64
	Duration  time.Duration
	Error     error
	Hostname  string
	Time      time.Duration
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

// TLSHandshakeEvent is emitted when conn.Handshake returns.
type TLSHandshakeEvent struct {
	Config          TLSConfig
	ConnectionState TLSConnectionState
	ConnID          int64
	Duration        time.Duration
	Error           error
	Time            time.Duration
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
	Close                   *CloseEvent                   `json:",omitempty"`
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
	Resolve                 *ResolveEvent                 `json:",omitempty"`
	TLSHandshake            *TLSHandshakeEvent            `json:",omitempty"`
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
