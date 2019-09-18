// Package dnsconf allows to configure a DNS resolver
package dnsconf

import (
	"errors"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/doh"
	"github.com/bassosimone/netx/internal/dopot"
	"github.com/bassosimone/netx/internal/dopou"
	"github.com/bassosimone/netx/internal/dot"
)

// Do implements netx.Dialer.ConfigureDNS.
func Do(dialer *dialerapi.Dialer, network, address string) error {
	if network == "doh" {
		clnt, err := doh.NewClient(dialer, address)
		if err == nil {
			dialer.LookupHost = clnt.NewResolver().LookupHost
		}
		return err
	}
	if network == "dot" {
		clnt, err := dot.NewClient(dialer, address)
		if err != nil {
			return err
		}
		dialer.LookupHost = clnt.NewResolver().LookupHost
		return nil
	}
	if network == "tcp" {
		resolver := dopot.NewResolver(dialer, address)
		dialer.LookupHost = resolver.LookupHost
		return nil
	}
	if network == "udp" {
		resolver := dopou.NewResolver(dialer, address)
		dialer.LookupHost = resolver.LookupHost
		return nil
	}
	return errors.New("dnsconf: unsupported network value")
}
