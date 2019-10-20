// Package ootlshandshaker contains OONI's TLS handshaker
package ootlshandshaker

import (
	"context"
	"crypto/tls"
	"net"
)

// Handshaker is OONI's TLS handshaker
type Handshaker struct{}

// New creates a new OONI TLS handshaker
func New() *Handshaker {
	return new(Handshaker)
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
	config = config.Clone()
	if config.ServerName == "" {
		config.ServerName = domain
	}
	tlsconn := tls.Client(conn, config)
	errch := make(chan error, 1)
	go func() {
		errch <- tlsconn.Handshake()
	}()
	select {
	case err := <-errch:
		return tlsconn, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
