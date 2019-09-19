package godns_test

import (
	"context"
	"testing"
	"time"

	"github.com/bassosimone/netx/internal/dnstransport/dnsoverhttps"
	"github.com/bassosimone/netx/internal/godns"
	"github.com/bassosimone/netx/internal/testingx"
)

func TestIntegrationSuccess(t *testing.T) {
	start := time.Now()
	transport := dnsoverhttps.NewTransport(
		start, testingx.StdoutHandler,
		"https://cloudflare-dns.com/dns-query",
	)
	client := godns.NewClient(start, testingx.StdoutHandler, transport)
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
		start, testingx.StdoutHandler,
		"https://cloudflare-dns.com/dns-query",
	)
	conn := godns.NewPseudoConn(start, testingx.StdoutHandler, transport)
	err := conn.SetDeadline(time.Now()) // very short deadline
	reply := make([]byte, 1<<17)
	n, err := conn.Read(reply)
	if err == nil {
		t.Fatal("expected to see an error here")
	}
	if n != 0 {
		t.Fatal("expected to see zero bytes here")
	}
}
