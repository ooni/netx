// Package dnsconf allows to configure a DNS resolver
package dnsconf

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ooni/netx/dnsx"
	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/internal/dnsclient/emittingdnsclient"
	"github.com/ooni/netx/internal/dnsclient/oodnsclient"
	"github.com/ooni/netx/internal/dnstransport/dnsoverhttps"
	"github.com/ooni/netx/internal/dnstransport/dnsovertcp"
	"github.com/ooni/netx/internal/dnstransport/dnsoverudp"
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
	// Logic to make sure we'll use the dialer in the new HTTP transport. We have
	// an already well configured config that works for http2 (as explained in a
	// comment there). Here we just use it because it's what we need.
	dialer.TLSConfig = transport.TLSClientConfig
	// Arrange the configuration such that we always use `dialer` for dialing.
	transport.Dial = dialer.Dial
	transport.DialContext = dialer.DialContext
	transport.DialTLS = dialer.DialTLS
	transport.MaxConnsPerHost = 1 // seems to be better for cloudflare DNS
	return &http.Client{Transport: transport}
}

func withPort(address, port string) string {
	// Handle the case where port was not specified. We have written in
	// a bunch of places that we can just pass a domain in this case and
	// so we need to gracefully ensure this is still possible.
	_, _, err := net.SplitHostPort(address)
	if err != nil && strings.Contains(err.Error(), "missing port in address") {
		address = net.JoinHostPort(address, port)
	}
	return address
}

// NewResolver returns a new resolver using this Dialer as dialer for
// creating new network connections used for resolving.
func NewResolver(
	dialer *dialerapi.Dialer, network, address string,
) (dnsx.Client, error) {
	// Implementation note: system dns goes first because doesn't have transport
	if network == "system" {
		return emittingdnsclient.New(&net.Resolver{
			PreferGo: false,
		}), nil
	}
	var transport dnsx.RoundTripper
	if network == "doh" {
		transport = dnsoverhttps.NewTransport(
			newHTTPClientForDoH(dialer.Beginning, dialer.Handler), address,
		)
	} else if network == "dot" {
		transport = dnsovertcp.NewTransport(
			// We need a child dialer here to avoid an endless loop where the
			// dialer will ask us to resolve, we'll tell the dialer to dial, it
			// will ask us to resolve, ...
			dialerapi.NewDialer(dialer.Beginning, dialer.Handler).DialTLSContext,
			withPort(address, "853"),
		)
	} else if network == "tcp" {
		transport = dnsovertcp.NewTransport(
			// Same rationale as above: avoid possible endless loop
			dialerapi.NewDialer(dialer.Beginning, dialer.Handler).DialContext,
			withPort(address, "53"),
		)
	} else if network == "udp" {
		transport = dnsoverudp.NewTransport(
			// Same rationale as above: avoid possible endless loop
			dialerapi.NewDialer(dialer.Beginning, dialer.Handler).DialContext,
			withPort(address, "53"),
		)
	}
	if transport == nil {
		return nil, errors.New("dnsconf: unsupported network value")
	}
	return emittingdnsclient.New(oodnsclient.New(transport)), nil
}
