// Package tlsx contains crypto/tls extensions
package tlsx

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"time"

	"github.com/ooni/netx/model"
)

// ReadCABundle read a CA bundle from file
func ReadCABundle(path string) (*x509.CertPool, error) {
	cert, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(cert)
	return pool, nil
}

// EmitTLSHandshakeEvent emits the TLSHandshakeEvent.
func EmitTLSHandshakeEvent(
	handler model.Handler,
	state tls.ConnectionState,
	elapsed time.Duration,
	operationDuration time.Duration,
	err error,
	config *tls.Config,
) {
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
			Duration: operationDuration,
			Error:    err,
			Time:     elapsed,
		},
	})
}

func simplifyCerts(in []*x509.Certificate) (out []model.X509Certificate) {
	for _, cert := range in {
		out = append(out, model.X509Certificate{
			Data: cert.Raw,
		})
	}
	return
}
