package resolver

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ooni/netx/internal/tracing"
)

func TestInvalidURL(t *testing.T) {
	network, address, err := ParseDNSConfigFromURL("\t")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if network != "" {
		t.Fatal("expected empty network here")
	}
	if address != "" {
		t.Fatal("expected empty address here")
	}
}

func TestNotImplemented(t *testing.T) {
	resolver, err := New(
		time.Now(), "antani", "",
		func(beginning time.Time) http.RoundTripper {
			return http.DefaultTransport
		},
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resolver != nil {
		t.Fatal("expected nil resolver here")
	}
}

func TestSystem(t *testing.T) {
	testResolver(t, "system:///")
}

func TestNetgo(t *testing.T) {
	testResolver(t, "netgo:///")
}

func TestUDP(t *testing.T) {
	testResolver(t, "udp://8.8.8.8/")
}

func TestUDPWithPort(t *testing.T) {
	testResolver(t, "udp://8.8.8.8:53/")
}

func TestTCP(t *testing.T) {
	testResolver(t, "tcp://8.8.8.8/")
}

func TestTCPWithPort(t *testing.T) {
	testResolver(t, "tcp://8.8.8.8:53/")
}

func TestDoT(t *testing.T) {
	testResolver(t, "dot://1.1.1.1/")
}

func TestDoTWithPort(t *testing.T) {
	testResolver(t, "dot://1.1.1.1:853/")
}

func TestDoTWithDomain(t *testing.T) {
	testResolver(t, "dot://dns.quad9.net/")
}

func TestDoTWithDomainAndPort(t *testing.T) {
	testResolver(t, "dot://dns.quad9.net:853/")
}

func TestDoH(t *testing.T) {
	testResolver(t, "https://cloudflare-dns.com/dns-query")
}

func testResolver(t *testing.T, URL string) *tracing.Saver {
	network, address, err := ParseDNSConfigFromURL(URL)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := New(
		time.Now(), network, address,
		func(beginning time.Time) http.RoundTripper {
			return http.DefaultTransport
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	saver := tracing.NewSaver()
	ctx := tracing.WithHandler(context.Background(), saver)
	addrs, err := resolver.LookupHost(ctx, "www.kernel.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) < 1 {
		t.Fatal("too few results")
	}
	return saver
}
