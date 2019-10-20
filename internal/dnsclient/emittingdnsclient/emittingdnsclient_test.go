package emittingdnsclient

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/ooni/netx/internal/handlers/counthandler"
	"github.com/ooni/netx/internal/tracing"
)

func TestLookupAddr(t *testing.T) {
	client := New(&net.Resolver{})
	names, err := client.LookupAddr(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) < 0 {
		t.Fatal("no names returned")
	}
}

func TestLookupCNAME(t *testing.T) {
	client := New(&net.Resolver{})
	cname, err := client.LookupCNAME(context.Background(), "www.ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	if cname == "" {
		t.Fatal("no cname returned")
	}
}

func TestLookupHost(t *testing.T) {
	client := New(&net.Resolver{})
	addrs, err := client.LookupHost(context.Background(), "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) < 0 {
		t.Fatal("no addresses returned")
	}
}

func TestLookupHostWithTracing(t *testing.T) {
	client := New(&net.Resolver{})
	info := tracing.Info{
		Beginning: time.Now(),
		Handler:   &counthandler.Handler{},
	}
	ctx := tracing.WithInfo(context.Background(), &info)
	addrs, err := client.LookupHost(ctx, "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) < 0 {
		t.Fatal("no addresses returned")
	}
	if info.Handler.(*counthandler.Handler).Count == 0 {
		t.Fatal("no events emitted")
	}
}

func TestLookupMX(t *testing.T) {
	client := New(&net.Resolver{})
	records, err := client.LookupMX(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) < 0 {
		t.Fatal("no addresses returned")
	}
}

func TestLookupNS(t *testing.T) {
	client := New(&net.Resolver{})
	records, err := client.LookupNS(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) < 0 {
		t.Fatal("no addresses returned")
	}
}
