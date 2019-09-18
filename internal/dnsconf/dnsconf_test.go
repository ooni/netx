package dnsconf_test

import (
	"testing"
	"time"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dnsconf"
	"github.com/bassosimone/netx/internal/testingx"
)

func TestIntegrationNewResolver(t *testing.T) {
	d := dialerapi.NewDialer(time.Now(), testingx.StdoutHandler)
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

func TestIntegrationDo(t *testing.T) {
	d := dialerapi.NewDialer(time.Now(), testingx.StdoutHandler)
	err := dnsconf.Do(d, "dot", "dns.quad9.net")
	if err != nil {
		t.Fatal(err)
	}
}
