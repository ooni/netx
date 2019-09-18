package oodns

import (
	"context"
	"testing"
	"time"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dot"
	"github.com/bassosimone/netx/internal/testingx"
)

func TestIntegration(t *testing.T) {
	dialer := dialerapi.NewDialer(time.Now(), testingx.StdoutHandler)
	dotclient, err := dot.NewClient(dialer, "dns.quad9.net")
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(testingx.StdoutHandler, dotclient)
	addrs, err := client.LookupHost(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		t.Log(addr)
	}
}
