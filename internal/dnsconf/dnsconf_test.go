package dnsconf_test

import (
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/internal/dnsconf"
)

func TestIntegrationNewResolver(t *testing.T) {
	d := dialerapi.NewDialer(time.Now(), handlers.NoHandler)
	resolver, err := dnsconf.NewResolver(
		d, "udp", "8.8.8.8:53",
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolver == nil {
		t.Fatal("expected non-nil resolver here")
	}

	resolver, err = dnsconf.NewResolver(
		d, "tcp", "8.8.8.8:53",
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolver == nil {
		t.Fatal("expected non-nil resolver here")
	}

	resolver, err = dnsconf.NewResolver(
		d, "dot", "dns.quad9.net",
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolver == nil {
		t.Fatal("expected non-nil resolver here")
	}

	resolver, err = dnsconf.NewResolver(
		d, "doh", "https://cloudflare-dns.com/dns-query",
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolver == nil {
		t.Fatal("expected non-nil resolver here")
	}

	resolver, err = dnsconf.NewResolver(
		d, "antani", "https://cloudflare-dns.com/dns-query",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resolver != nil {
		t.Fatal("expected a nil resolver here")
	}
}

func TestIntegrationNewResolverBadTCPEndpoint(t *testing.T) {
	d := dialerapi.NewDialer(time.Now(), handlers.NoHandler)
	resolver, err := dnsconf.NewResolver(
		d, "tcp", "8.8.8.8",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resolver != nil {
		t.Fatal("expected a nil resolver here")
	}
}

func TestIntegrationDo(t *testing.T) {
	d := dialerapi.NewDialer(time.Now(), handlers.NoHandler)
	err := dnsconf.ConfigureDNS(d, "dot", "dns.quad9.net")
	if err != nil {
		t.Fatal(err)
	}
}
