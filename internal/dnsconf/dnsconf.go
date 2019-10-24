// Package dnsconf allows to configure a DNS resolver
package dnsconf

import (
	"github.com/ooni/netx/internal"
	"github.com/ooni/netx/internal/resolver"
)

// ConfigureDNS implements netx.Dialer.ConfigureDNS.
func ConfigureDNS(dialer *internal.Dialer, network, address string) error {
	r, err := resolver.New(dialer.Beginning, dialer.Handler, network, address)
	if err == nil {
		dialer.Resolver = r
	}
	return err
}
