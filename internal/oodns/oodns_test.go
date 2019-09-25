package oodns_test

import (
	"context"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/dnstransport/dnsovertcp"
	"github.com/ooni/netx/internal/oodns"
)

func TestIntegration(t *testing.T) {
	client := oodns.NewClient(
		handlers.NoHandler, dnsovertcp.NewTransport(
			time.Now(), handlers.NoHandler, "dns.quad9.net",
		),
	)
	addrs, err := client.LookupHost(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		t.Log(addr)
	}
}
