package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
)

func testresolverquick(t *testing.T, network, address string) {
	resolver, err := New(time.Now(), handlers.NoHandler, network, address)
	if err != nil {
		t.Fatal(err)
	}
	if resolver == nil {
		t.Fatal("expected non-nil resolver here")
	}
	addrs, err := resolver.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if addrs == nil {
		t.Fatal("expected non-nil addrs here")
	}
	var foundquad8 bool
	for _, addr := range addrs {
		if addr == "8.8.8.8" {
			foundquad8 = true
		}
	}
	if !foundquad8 {
		t.Fatal("did not find 8.8.8.8 in ouput")
	}
}

func TestIntegrationNewResolverUDPAddress(t *testing.T) {
	testresolverquick(t, "udp", "8.8.8.8:53")
}

func TestIntegrationNewResolverUDPAddressNoPort(t *testing.T) {
	testresolverquick(t, "udp", "8.8.8.8")
}

func TestIntegrationNewResolverUDPDomain(t *testing.T) {
	testresolverquick(t, "udp", "dns.google.com:53")
}

func TestIntegrationNewResolverUDPDomainNoPort(t *testing.T) {
	testresolverquick(t, "udp", "dns.google.com")
}

func TestIntegrationNewResolverSystem(t *testing.T) {
	testresolverquick(t, "system", "")
}

func TestIntegrationNewResolverTCPAddress(t *testing.T) {
	testresolverquick(t, "tcp", "8.8.8.8:53")
}

func TestIntegrationNewResolverTCPAddressNoPort(t *testing.T) {
	testresolverquick(t, "tcp", "8.8.8.8")
}

func TestIntegrationNewResolverTCPDomain(t *testing.T) {
	testresolverquick(t, "tcp", "dns.google.com:53")
}

func TestIntegrationNewResolverTCPDomainNoPort(t *testing.T) {
	testresolverquick(t, "tcp", "dns.google.com")
}

func TestIntegrationNewResolverDoTAddress(t *testing.T) {
	testresolverquick(t, "dot", "9.9.9.9:853")
}

func TestIntegrationNewResolverDoTAddressNoPort(t *testing.T) {
	testresolverquick(t, "dot", "9.9.9.9")
}

func TestIntegrationNewResolverDoTDomain(t *testing.T) {
	testresolverquick(t, "dot", "dns.quad9.net:853")
}

func TestIntegrationNewResolverDoTDomainNoPort(t *testing.T) {
	testresolverquick(t, "dot", "dns.quad9.net")
}

func TestIntegrationNewResolverDoH(t *testing.T) {
	testresolverquick(t, "doh", "https://cloudflare-dns.com/dns-query")
}

func TestIntegrationNewResolverInvalid(t *testing.T) {
	resolver, err := New(
		time.Now(), handlers.StdoutHandler,
		"antani", "https://cloudflare-dns.com/dns-query",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resolver != nil {
		t.Fatal("expected a nil resolver here")
	}
}
