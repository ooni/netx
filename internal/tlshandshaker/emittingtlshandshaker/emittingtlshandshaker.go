// Package emittingtlshandshaker contains an event-emitting TLS handshaker
package emittingtlshandshaker

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/ooni/netx/internal/tlshandshaker"
	"github.com/ooni/netx/internal/tracing"
)

// Handshaker is the event emitting TLS handshaker
type Handshaker struct {
	handshaker tlshandshaker.Model
}

// New creates a new OONI TLS handshaker
func New(handshaker tlshandshaker.Model) *Handshaker {
	return &Handshaker{handshaker: handshaker}
}

// Do creates and handshakes a TLS connection. In case of successful
// handshake, returns the connection and a nil error. In case of error
// during the handshake (e.g. invalid certificate), returns both a
// connection and an error. In case of context timeout, returns instead
// a nil connection and an error.
func (h *Handshaker) Do(
	ctx context.Context, conn net.Conn,
	config *tls.Config, domain string,
) (*tls.Conn, error) {
	if info := tracing.ContextInfo(ctx); info != nil {
		info.EmitTLSHandshakeStart(config)
	}
	tlsconn, err := h.handshaker.Do(ctx, conn, config, domain)
	if info := tracing.ContextInfo(ctx); info != nil {
		var csp *tls.ConnectionState
		if tlsconn != nil {
			connstate := tlsconn.ConnectionState()
			csp = &connstate
		}
		info.EmitTLSHandshakeDone(csp, err)
	}
	return tlsconn, err
}
