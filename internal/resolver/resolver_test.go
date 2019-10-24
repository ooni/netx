package resolver

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/model"
)

func testresolverquick(t *testing.T, resolver model.DNSResolver) {
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
	testresolverquick(t, NewResolverUDP(
		time.Now(), handlers.NoHandler, new(net.Dialer), "8.8.8.8:53"))
}

func TestIntegrationNewResolverUDPDomain(t *testing.T) {
	testresolverquick(t, NewResolverUDP(
		time.Now(), handlers.NoHandler, new(net.Dialer), "dns.google.com:53"))
}

func TestIntegrationNewResolverTCPAddress(t *testing.T) {
	testresolverquick(t, NewResolverTCP(
		time.Now(), handlers.NoHandler, new(net.Dialer), "8.8.8.8:53"))
}

func TestIntegrationNewResolverTCPDomain(t *testing.T) {
	testresolverquick(t, NewResolverTCP(
		time.Now(), handlers.NoHandler, new(net.Dialer), "dns.google.com:53"))
}

func TestIntegrationNewResolverDoTAddress(t *testing.T) {
	testresolverquick(t, NewResolverTLS(
		time.Now(), handlers.NoHandler, &tlsdialer{}, "9.9.9.9:853"))
}

func TestIntegrationNewResolverDoTDomain(t *testing.T) {
	testresolverquick(t, NewResolverTLS(
		time.Now(), handlers.NoHandler, &tlsdialer{}, "dns.quad9.net:853"))
}

func TestIntegrationNewResolverDoH(t *testing.T) {
	testresolverquick(t, NewResolverHTTPS(
		time.Now(), handlers.NoHandler, http.DefaultClient,
		"https://cloudflare-dns.com/dns-query"))
}

type tlsdialer struct{}

func (*tlsdialer) DialTLS(network, address string) (net.Conn, error) {
	return tls.Dial(network, address, new(tls.Config))
}

func (*tlsdialer) DialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	return tls.Dial(network, address, new(tls.Config))
}
