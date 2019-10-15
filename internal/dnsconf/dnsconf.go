// Package dnsconf allows to configure a DNS resolver
package dnsconf

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/ooni/netx/dnsx"
	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/internal/dnstransport/dnsoverhttps"
	"github.com/ooni/netx/internal/dnstransport/dnsovertcp"
	"github.com/ooni/netx/internal/dnstransport/dnsoverudp"
	"github.com/ooni/netx/internal/godns"
	"github.com/ooni/netx/internal/httptransport"
	"github.com/ooni/netx/model"
)

// ConfigureDNS implements netx.Dialer.ConfigureDNS.
func ConfigureDNS(dialer *dialerapi.Dialer, network, address string) error {
	r, err := NewResolver(dialer, network, address)
	if err == nil {
		dialer.LookupHost = r.LookupHost
	}
	return err
}

func newHTTPClientForDoH(beginning time.Time, handler model.Handler) *http.Client {
	dialer := dialerapi.NewDialer(beginning, handler)
	transport := httptransport.NewTransport(dialer.Beginning, dialer.Handler)
	// Logic to make sure we'll use the dialer in the new HTTP transport
	dialer.TLSConfig = transport.TLSClientConfig
	transport.Dial = dialer.Dial
	transport.DialContext = dialer.DialContext
	transport.DialTLS = dialer.DialTLS
	transport.MaxConnsPerHost = 1 // seems to be better for cloudflare DNS
	return &http.Client{Transport: transport}
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
			newHTTPClientForDoH(dialer.Beginning, dialer.Handler), address,
		)
	} else if network == "dot" {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			transport = dnsovertcp.NewTransport(
				dialer.Beginning, dialer.Handler, address,
			)
		} else {
			dotTransport := dnsovertcp.NewTransport(
				dialer.Beginning, dialer.Handler, host,
			)
			dotTransport.Port = port
			transport = dotTransport
		}
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
