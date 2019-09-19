// Package dnsconf allows to configure a DNS resolver
package dnsconf

import (
	"errors"
	"net"

	"github.com/bassosimone/netx/dnsx"
	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dnstransport/dnsoverhttps"
	"github.com/bassosimone/netx/internal/dnstransport/dnsovertcp"
	"github.com/bassosimone/netx/internal/dnstransport/dnsoverudp"
	"github.com/bassosimone/netx/internal/godns"
)

// ConfigureDNS implements netx.Dialer.ConfigureDNS.
func ConfigureDNS(dialer *dialerapi.Dialer, network, address string) error {
	r, err := NewResolver(dialer, network, address)
	if err == nil {
		dialer.LookupHost = r.LookupHost
	}
	return err
}

// NewResolver returns a new resolver using this Dialer as dialer for
// creating new network connections used for resolving.
func NewResolver(dialer *dialerapi.Dialer, network, address string) (*net.Resolver, error) {
	var transport dnsx.RoundTripper
	if network == "doh" {
		transport = dnsoverhttps.NewTransport(
			dialer.Beginning, dialer.Handler, address,
		)
	} else if network == "dot" {
		transport = dnsovertcp.NewTransport(
			dialer.Beginning, dialer.Handler, address,
		)
	} else if network == "tcp" {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		dotTransport := dnsovertcp.NewTransport(
			dialer.Beginning, dialer.Handler, host,
		)
		dotTransport.Port = port
		dotTransport.NoTLS = true
		transport = dotTransport
	} else if network == "udp" {
		transport = dnsoverudp.NewTransport(
			dialer.Beginning, dialer.Handler, address,
		)
	}
	if transport == nil {
		return nil, errors.New("dnsconf: unsupported network value")
	}
	return godns.NewClient(dialer.Beginning, dialer.Handler, transport), nil
}
