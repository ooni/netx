// Package dnsconf allows to configure a DNS resolver
package dnsconf

import (
	"errors"
	"net"

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
func NewResolver(dialer *dialerapi.Dialer, network, address string) (r *net.Resolver, err error) {
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
		var clnt *dopot.Client
		clnt, err = dopot.NewClient(dialer, address)
		if err == nil {
			r = clnt.NewResolver()
		}
		return
	}
	if network == "udp" {
		var clnt *dopou.Client
		clnt, err = dopou.NewClient(dialer, address)
		if err == nil {
			r = clnt.NewResolver()
		}
		return
	}
	return nil, errors.New("dnsconf: unsupported network value")
}
