// Package emittingtlshandshaker contains an event-emitting TLS handshaker
package emittingtlshandshaker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"time"

	"github.com/ooni/netx/internal/tlshandshaker"
	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
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
	start := time.Now()
	tlsconn, err := h.handshaker.Do(ctx, conn, config, domain)
	stop := time.Now()
	if info := tracing.ContextInfo(ctx); info != nil && tlsconn != nil {
		connstate := tlsconn.ConnectionState()
		info.Handler.OnMeasurement(model.Measurement{
			TLSHandshake: &model.TLSHandshakeEvent{
				Config: model.TLSConfig{
					NextProtos: config.NextProtos,
					ServerName: config.ServerName,
				},
				ConnectionState: model.TLSConnectionState{
					CipherSuite:                connstate.CipherSuite,
					NegotiatedProtocol:         connstate.NegotiatedProtocol,
					NegotiatedProtocolIsMutual: connstate.NegotiatedProtocolIsMutual,
					PeerCertificates:           simplify(connstate.PeerCertificates),
					Version:                    connstate.Version,
				},
				Duration: stop.Sub(start),
				Error:    err,
				ConnID:   info.ConnID,
				Time:     stop.Sub(info.Beginning),
			},
		})
	}
	return tlsconn, err
}

func simplify(in []*x509.Certificate) (out []model.X509Certificate) {
	for _, cert := range in {
		out = append(out, model.X509Certificate{
			Data: cert.Raw,
		})
	}
	return
}
