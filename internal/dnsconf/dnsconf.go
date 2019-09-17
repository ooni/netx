// Package dnsconf allows to configure a DNS resolver
package dnsconf

import (
	"errors"
	"net"
	"net/url"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/doh"
	"github.com/bassosimone/netx/internal/dopot"
	"github.com/bassosimone/netx/internal/dopou"
	"github.com/bassosimone/netx/internal/dot"
)

// Do implements netx.Dialer.ConfigureDNS.
func Do(dialer *dialerapi.Dialer, network, address string) error {
	if network == "doh" {
		URL, err := url.Parse(address)
		if err != nil {
			return err
		}
		resolver := doh.NewResolver(dialer, URL)
		dialer.LookupHost = resolver.LookupHost
		return nil
	}
	if network == "dot" {
		first, err := lookupFirstHost(address)
		if err != nil {
			return err
		}
		resolver := dot.NewResolver(dialer, first, address)
		dialer.LookupHost = resolver.LookupHost
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

func lookupFirstHost(address string) (string, error) {
	addrs, err := net.LookupHost(address)
	if err != nil {
		return "", err
	}
	if len(addrs) < 1 {
		return "", errors.New("dnsconf: net.LookupHost returned an empty slice")
	}
	return addrs[0], nil
}
