package dnsconf_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/internal/dnsconf"
	"github.com/ooni/netx/model"
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
		d, "system", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolver == nil {
		t.Fatal("expected non-nil resolver here")
	}

	resolver, err = dnsconf.NewResolver(
		d, "godns", "",
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
		d, "dot", "1.1.1.1:853",
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

func TestIntegrationGoDNSResolverSuccess(t *testing.T) {
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
	addrs, err := resolver.LookupHost(context.Background(), "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) < 1 {
		t.Fatal("expected non empty addrs here")
	}
}

func TestIntegrationGoDNSResolverFailure(t *testing.T) {
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
		ctx context.Context, handler model.Handler,
		network, onlyhost, onlyport string, connid int64,
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
