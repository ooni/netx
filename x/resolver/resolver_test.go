package resolver

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
)

func TestIntegrationSucces(t *testing.T) {
	reso := newresolver()
	addrs, err := reso.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) < 1 {
		t.Fatal("too few addresses")
	}
}

func TestIntegrationContextTimeout(t *testing.T) {
	reso := newresolver()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Microsecond)
	defer cancel()
	addrs, err := reso.LookupHost(ctx, "dns.google.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatal("unexpected error type")
	}
	if len(addrs) > 0 {
		t.Fatal("too little addresses")
	}
}

func newresolver() *Resolver {
	return New(time.Now(), handlers.NoHandler, new(net.Resolver))
}
