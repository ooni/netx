// Package tlshandshaker contains the generic tls handshaker model
package tlshandshaker

import (
	"context"
	"crypto/tls"
	"net"
)

// Model is the model for all TLS handshakers
type Model interface {
	Do(ctx context.Context, conn net.Conn,
		config *tls.Config, domain string) (*tls.Conn, error)
}
