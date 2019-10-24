// Package dialer contains the dialer's API. The dialer defined
// in here implements basic DNS, but that is overridable.
package dialer

import (
	"crypto/tls"
	"time"

	"github.com/ooni/netx/internal/dialer/dnsdialer"
	"github.com/ooni/netx/internal/dialer/tlsdialer"
	"github.com/ooni/netx/model"
)

// New creates a new model.Dialer
func New(
	beginning time.Time,
	handler model.Handler,
	resolver model.DNSResolver,
	dialer model.Dialer,
) *dnsdialer.Dialer {
	return dnsdialer.New(
		beginning, handler, resolver, dialer,
	)
}

// NewTLS creates a new model.TLSDialer
func NewTLS(
	beginning time.Time,
	handler model.Handler,
	dialer model.Dialer,
	config *tls.Config,
) *tlsdialer.TLSDialer {
	return tlsdialer.New(
		beginning, handler, dialer, config,
	)
}
