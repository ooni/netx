package dnsconf_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ooni/netx/dnsx"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/internal/dnsconf"
)

func testresolverquick(t *testing.T, network, address string) {
	var resolver dnsx.Client
	d := dialerapi.NewDialer(time.Now(), handlers.NoHandler)
	resolver, err := dnsconf.NewResolver(d, network, address)
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

func TestIntegrationNewResolverGoDNS(t *testing.T) {
	testresolverquick(t, "godns", "")
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
	d := dialerapi.NewDialer(time.Now(), handlers.NoHandler)
	resolver, err := dnsconf.NewResolver(
		d, "antani", "https://cloudflare-dns.com/dns-query",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resolver != nil {
		t.Fatal("expected a nil resolver here")
	}
}

func testconfigurednsquick(t *testing.T, network, address string) {
	d := dialerapi.NewDialer(time.Now(), handlers.NoHandler)
	err := dnsconf.ConfigureDNS(d, network, address)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := d.DialTLS("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn here")
	}
	conn.Close()
}

func TestGoDNSDialContextExFailure(t *testing.T) {
	d := dialerapi.NewDialer(time.Now(), handlers.NoHandler)
	resolver, err := dnsconf.NewResolver(
		d, "godns", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolver == nil {
		t.Fatal("expected non-nil resolver here")
	}
	// Override the function used to established a connection so to return
	// an error. This will cause the LookupHost to fail and allows us to
	// fully cover the codepath where net.Resolver.Dial invoked.
	d.DialHostPort = func(
		ctx context.Context, network, onlyhost, onlyport string, connid int64,
	) (*connx.MeasuringConn, error) {
		return nil, errors.New("mocked error")
	}
	addrs, err := resolver.LookupHost(context.Background(), "www.google.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if len(addrs) != 0 {
		t.Fatal("expected empty addrs here")
	}
}

func TestIntegrationConfigureDNSGoDNS(t *testing.T) {
	testconfigurednsquick(t, "godns", "")
}
