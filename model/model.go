// Package model contains our data model
package model

import (
	"net/http"
	"time"
)

// CloseEvent is emitted when a connection is closed.
type CloseEvent struct {
	ConnID   int64
	Duration time.Duration
	Error    error
	Time     time.Duration
}

// ConnectEvent is emitted when a connection is established.
type ConnectEvent struct {
	ConnID        int64
	Duration      time.Duration
	Error         error
	LocalAddress  string
	Network       string
	RemoteAddress string
	Time          time.Duration
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

// HTTPResponseDoneEvent is emitted after we have received the body.
type HTTPResponseDoneEvent struct {
	Time          time.Duration
	TransactionID int64
}

// ReadEvent is emitted when data is read.
type ReadEvent struct {
	ConnID   int64
	Duration time.Duration
	Error    error
	NumBytes int64
	Time     time.Duration
}

// ResolveEvent is emitted when a domain name is resolved.
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
	DERContext []byte
}

// TLSConnectionState contains the TLS connection state.
type TLSConnectionState struct {
	CipherSuite                uint16
	NegotiatedProtocol         string
	NegotiatedProtocolIsMutual bool
	PeerCertificates           []X509Certificate
	Version                    uint16
}

// TLSHandshakeEvent is emitted when a TLS handshake completes.
type TLSHandshakeEvent struct {
	Config          TLSConfig
	ConnectionState TLSConnectionState
	ConnID          int64
	Duration        time.Duration
	Error           error
	Time            time.Duration
}

// WriteEvent is emitted when data is written.
type WriteEvent struct {
	ConnID   int64
	Duration time.Duration
	Error    error
	NumBytes int64
	Time     time.Duration
}

// Measurement is a measurement.
type Measurement struct {
	Close                   *CloseEvent                   `json:",omitempty"`
	Connect                 *ConnectEvent                 `json:",omitempty"`
	HTTPConnectionReady     *HTTPConnectionReadyEvent     `json:",omitempty"`
	HTTPRequestStart        *HTTPRequestStartEvent        `json:",omitempty"`
	HTTPRequestHeadersDone  *HTTPRequestHeadersDoneEvent  `json:",omitempty"`
	HTTPRequestDone         *HTTPRequestDoneEvent         `json:",omitempty"`
	HTTPResponseStart       *HTTPResponseStartEvent       `json:",omitempty"`
	HTTPResponseHeadersDone *HTTPResponseHeadersDoneEvent `json:",omitempty"`
	HTTPResponseDone        *HTTPResponseDoneEvent        `json:",omitempty"`
	Read                    *ReadEvent                    `json:",omitempty"`
	Resolve                 *ResolveEvent                 `json:",omitempty"`
	TLSHandshake            *TLSHandshakeEvent            `json:",omitempty"`
	Write                   *WriteEvent                   `json:",omitempty"`
}
