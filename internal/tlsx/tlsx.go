// Package tlsx contains crypto/tls extensions
package tlsx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"time"

	"github.com/bassosimone/netx/internal/connx"
	"github.com/bassosimone/netx/model"
)

// Handshake performs a TLS handshake.
func Handshake(
	ctx context.Context, config *tls.Config, timeout time.Duration,
	conn *connx.MeasuringConn, handler model.Handler,
) (tc *tls.Conn, err error) {
	tc = tls.Client(net.Conn(conn), config)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ech := make(chan error, 1)
	start := time.Now()
	go func() {
		ech <- tc.Handshake()
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-ech:
		// FALLTHROUGH
	}
	stop := time.Now()
	state := tc.ConnectionState()
	handler.OnMeasurement(model.Measurement{
		TLSHandshake: &model.TLSHandshakeEvent{
			Config: model.TLSConfig{
				NextProtos: config.NextProtos,
				ServerName: config.ServerName,
			},
			ConnectionState: model.TLSConnectionState{
				CipherSuite:                state.CipherSuite,
				NegotiatedProtocol:         state.NegotiatedProtocol,
				NegotiatedProtocolIsMutual: state.NegotiatedProtocolIsMutual,
				PeerCertificates:           simplifyCerts(state.PeerCertificates),
				Version:                    state.Version,
			},
			Duration: stop.Sub(start),
			Error:    err,
			ConnID:   conn.ID,
			Time:     stop.Sub(conn.Beginning),
		},
	})
	if err != nil {
		tc.Close()
		tc = nil
	}
	return
}

func simplifyCerts(in []*x509.Certificate) (out []model.X509Certificate) {
	for _, cert := range in {
		out = append(out, model.X509Certificate{
			Data: cert.Raw,
		})
	}
	return
}
