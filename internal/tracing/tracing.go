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
	Beginning       time.Time
	ConnID          int64
	Creator         string
	Handler         model.Handler
	HTTPRoundTripID int64
	ResolveID       int64
}

// NewInfo creates a new Info instance
func NewInfo(creator string, beginning time.Time, handler model.Handler) *Info {
	return &Info{
		Beginning: beginning,
		Creator:   creator,
		Handler:   handler,
	}
}

// Clone clones Info
func (info *Info) Clone(creator string) *Info {
	return &Info{
		Beginning:       info.Beginning,
		ConnID:          info.ConnID,
		Creator:         creator,
		Handler:         info.Handler,
		HTTPRoundTripID: info.HTTPRoundTripID,
		ResolveID:       info.ResolveID,
	}
}

// CloneWithNewHTTPRoundTripID clones Info with new round-trip ID
func (info *Info) CloneWithNewHTTPRoundTripID(creator string, id int64) *Info {
	return &Info{
		Beginning:       info.Beginning,
		ConnID:          info.ConnID,
		Creator:         creator,
		Handler:         info.Handler,
		HTTPRoundTripID: id,
		ResolveID:       info.ResolveID,
	}
}

// CloneWithNewResolveID clones Info with new resolve ID
func (info *Info) CloneWithNewResolveID(creator string, id int64) *Info {
	return &Info{
		Beginning:       info.Beginning,
		ConnID:          info.ConnID,
		Creator:         creator,
		Handler:         info.Handler,
		HTTPRoundTripID: info.HTTPRoundTripID,
		ResolveID:       id,
	}
}

// CloneWithNewConnID clones Info with new conn ID
func (info *Info) CloneWithNewConnID(creator string, id int64) *Info {
	return &Info{
		Beginning:       info.Beginning,
		ConnID:          id,
		Creator:         creator,
		Handler:         info.Handler,
		HTTPRoundTripID: info.HTTPRoundTripID,
		ResolveID:       info.ResolveID,
	}
}

// BaseEvent creates a new base event
func (info *Info) BaseEvent() model.BaseEvent {
	return model.BaseEvent{
		ConnID:          info.ConnID,
		ElapsedTime:     time.Now().Sub(info.Beginning),
		HTTPRoundTripID: info.HTTPRoundTripID,
		ResolveID:       info.ResolveID,
	}
}

// EmitTLSHandshakeStart emits the TLSHandshakeStartEvent event
func (info *Info) EmitTLSHandshakeStart(config *tls.Config) {
	info.Handler.OnMeasurement(model.Measurement{
		TLSHandshakeStart: &model.TLSHandshakeStartEvent{
			BaseEvent: info.BaseEvent(),
			Config: model.TLSConfig{
				NextProtos: config.NextProtos,
				ServerName: config.ServerName,
			},
		},
	})
}

// EmitTLSHandshakeDone emits the TLSHandshakeStartDone event
func (info *Info) EmitTLSHandshakeDone(csp *tls.ConnectionState, err error) {
	info.Handler.OnMeasurement(model.Measurement{
		TLSHandshakeDone: &model.TLSHandshakeDoneEvent{
			BaseEvent:       info.BaseEvent(),
			ConnectionState: safeConnState(csp),
			Error:           err,
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
