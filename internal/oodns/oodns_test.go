package oodns_test

import (
	"context"
	"testing"
	"time"

	"github.com/bassosimone/netx/internal/dnstransport/dnsovertcp"
	"github.com/bassosimone/netx/internal/oodns"
	"github.com/bassosimone/netx/handlers"
)

func TestIntegration(t *testing.T) {
	client := oodns.NewClient(
		handlers.StdoutHandler, dnsovertcp.NewTransport(
			time.Now(), handlers.StdoutHandler, "dns.quad9.net",
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
