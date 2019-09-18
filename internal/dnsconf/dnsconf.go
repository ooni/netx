// Package dnsconf allows to configure a DNS resolver
package dnsconf

import (
	"errors"

	"github.com/bassosimone/netx/dnsx"
	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/doh"
	"github.com/bassosimone/netx/internal/dopot"
	"github.com/bassosimone/netx/internal/dopou"
	"github.com/bassosimone/netx/internal/dot"
)

// Do implements netx.Dialer.ConfigureDNS.
func Do(dialer *dialerapi.Dialer, network, address string) error {
	r, err := NewResolver(dialer, network, address)
	if err == nil {
		dialer.LookupHost = r.LookupHost
	}
	return err
}

// NewResolver returns a new resolver using this Dialer as dialer for
// creating new network connections used for resolving.
func NewResolver(dialer *dialerapi.Dialer, network, address string) (r dnsx.Resolver, err error) {
	if network == "doh" {
		var clnt *doh.Client
		clnt, err = doh.NewClient(dialer, address)
		if err == nil {
			r = clnt.NewResolver()
		}
		return
	}
	if network == "dot" {
		var clnt *dot.Client
		clnt, err = dot.NewClient(dialer, address)
		if err == nil {
			r = clnt.NewResolver()
		}
		return
	}
	if network == "tcp" {
		return dopot.NewResolver(dialer, address), nil
	}
	if network == "udp" {
		return dopou.NewResolver(dialer, address), nil
	}
	return nil, errors.New("dnsconf: unsupported network value")
}
