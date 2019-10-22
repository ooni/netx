// Package tracing allows to trace events.
package tracing

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/ooni/netx/model"
)

type contextkey struct{}

// Info contains information useful for tracing
type Info struct {
	Beginning     time.Time
	ConnID        int64
	Handler       model.Handler
	TransactionID int64
}

// EmitTLSHandshakeStart emits the TLSHandshakeStartEvent event
func (info *Info) EmitTLSHandshakeStart(config *tls.Config) {
	info.Handler.OnMeasurement(model.Measurement{
		TLSHandshakeStart: &model.TLSHandshakeStartEvent{
			ConnID: info.ConnID,
			Config: model.TLSConfig{
				NextProtos: config.NextProtos,
				ServerName: config.ServerName,
			},
			Time: time.Now().Sub(info.Beginning),
		},
	})
}

// EmitTLSHandshakeDone emits the TLSHandshakeStartDone event
func (info *Info) EmitTLSHandshakeDone(csp *tls.ConnectionState, err error) {
	info.Handler.OnMeasurement(model.Measurement{
		TLSHandshakeDone: &model.TLSHandshakeDoneEvent{
			ConnID:          info.ConnID,
			ConnectionState: safeConnState(csp),
			Error:           err,
			Time:            time.Now().Sub(info.Beginning),
		},
	})
}

func safeConnState(csp *tls.ConnectionState) (out *model.TLSConnectionState) {
	if csp != nil {
		out = &model.TLSConnectionState{
			CipherSuite:                csp.CipherSuite,
			NegotiatedProtocol:         csp.NegotiatedProtocol,
			NegotiatedProtocolIsMutual: csp.NegotiatedProtocolIsMutual,
			PeerCertificates:           simplify(csp.PeerCertificates),
			Version:                    csp.Version,
		}
	}
	return
}

func simplify(in []*x509.Certificate) (out []model.X509Certificate) {
	for _, cert := range in {
		out = append(out, model.X509Certificate{
			Data: cert.Raw,
		})
	}
	return
}

// WithInfo returns a copy of ctx with the specific tracing info
func WithInfo(ctx context.Context, info *Info) context.Context {
	if info == nil {
		panic("nil handler") // like httptrace.WithClientTrace
	}
	return context.WithValue(ctx, contextkey{}, info)
}

// ContextInfo returns the trace info with the context.
func ContextInfo(ctx context.Context) *Info {
	ip, _ := ctx.Value(contextkey{}).(*Info)
	return ip
}
