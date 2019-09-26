// Package dnsconf allows to configure a DNS resolver
package dnsconf

import (
	"context"
	"errors"
	"net"

	"github.com/ooni/netx/dnsx"
	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/internal/dnstransport/dnsoverhttps"
	"github.com/ooni/netx/internal/dnstransport/dnsovertcp"
	"github.com/ooni/netx/internal/dnstransport/dnsoverudp"
	"github.com/ooni/netx/internal/godns"
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
func NewResolver(
	dialer *dialerapi.Dialer, network, address string,
) (*net.Resolver, error) {
	// Implementation note: system and godns need to be dealt with
	// separately because they don't have any transport.
	if network == "system" {
		return &net.Resolver{
			PreferGo: false,
		}, nil
	} else if network == "godns" {
		return &net.Resolver{
			PreferGo: true,
			Dial: func(
				ctx context.Context, network, address string,
			) (net.Conn, error) {
				conn, _, _, err := dialer.DialContextEx(ctx, network, address, false)
				if err != nil {
					return nil, err
				}
				// convince Go this is really a net.PacketConn
				return &connx.DNSMeasuringConn{MeasuringConn: *conn}, nil
			},
		}, nil
	} else {
		// FALLTHROUGH
	}
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
