package godns_test

import (
	"context"
	"testing"
	"time"

	"github.com/bassosimone/netx/handlers"
	"github.com/bassosimone/netx/internal/dnstransport/dnsoverhttps"
	"github.com/bassosimone/netx/internal/godns"
)

func TestIntegrationSuccess(t *testing.T) {
	start := time.Now()
	transport := dnsoverhttps.NewTransport(
		start, handlers.NoHandler,
		"https://cloudflare-dns.com/dns-query",
	)
	client := godns.NewClient(start, handlers.NoHandler, transport)
	addrs, err := client.LookupHost(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) < 1 {
		t.Fatal("expected at least one address")
	}
}

func TestIntegrationReadWithTimeout(t *testing.T) {
	start := time.Now()
	transport := dnsoverhttps.NewTransport(
		start, handlers.NoHandler,
		"https://cloudflare-dns.com/dns-query",
	)
	conn := godns.NewPseudoConn(start, handlers.NoHandler, transport)
	err := conn.SetDeadline(time.Now()) // very short deadline
	if err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, 1<<17)
	n, err := conn.Read(reply)
	if err == nil {
		t.Fatal("expected to see an error here")
	}
	if n != 0 {
		t.Fatal("expected to see zero bytes here")
	}
}
