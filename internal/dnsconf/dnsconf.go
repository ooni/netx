// Package dnsconf allows to configure a DNS resolver
package dnsconf

import (
	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/internal/resolver"
)

// ConfigureDNS implements netx.Dialer.ConfigureDNS.
func ConfigureDNS(dialer *dialerapi.Dialer, network, address string) error {
	r, err := resolver.New(dialer.Beginning, dialer.Handler, network, address)
	if err == nil {
		dialer.LookupHost = r.LookupHost
	}
	return err
}
