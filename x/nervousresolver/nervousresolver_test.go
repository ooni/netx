package nervousresolver

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestIntegrationBogon(t *testing.T) {
	resolver := New(
		&fakeresolverbogon{
			Resolver: new(net.Resolver),
			reply:    []string{"10.10.11.10"},
		},
		new(net.Resolver),
	)
	addrs, err := resolver.LookupHost(context.Background(), "www.kernel.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) < 1 {
		t.Fatal("expected an address here")
	}
	if resolver.bogonsCount != 1 {
		t.Fatal("unexpected number of bogons seen")
	}
}

func TestIntegrationMixed(t *testing.T) {
	resolver := New(
		&fakeresolverbogon{
			Resolver: new(net.Resolver),
			reply:    []string{"10.10.11.10", "8.8.8.8"},
		},
		new(net.Resolver),
	)
	addrs, err := resolver.LookupHost(context.Background(), "www.kernel.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) < 1 {
		t.Fatal("expected an address here")
	}
	if resolver.bogonsCount != 1 {
		t.Fatal("unexpected number of bogons seen")
	}
}

func TestIntegrationGood(t *testing.T) {
	resolver := New(
		&fakeresolverbogon{
			Resolver: new(net.Resolver),
			reply:    []string{"8.8.8.8"},
		},
		new(net.Resolver),
	)
	addrs, err := resolver.LookupHost(context.Background(), "www.kernel.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) < 1 {
		t.Fatal("expected an address here")
	}
	if resolver.bogonsCount != 0 {
		t.Fatal("unexpected number of bogons seen")
	}
}

func TestIntegrationAnotherError(t *testing.T) {
	resolver := New(
		&fakeresolverbogon{
			Resolver: new(net.Resolver),
			err:      errors.New("mocked error"),
		},
		new(net.Resolver),
	)
	addrs, err := resolver.LookupHost(context.Background(), "www.kernel.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) < 1 {
		t.Fatal("expected an address here")
	}
	if resolver.bogonsCount != 0 {
		t.Fatal("unexpected number of bogons seen")
	}
}

func TestOtherLookupMethods(t *testing.T) {
	// quick because I'm just composing
	resolver := New(new(net.Resolver), new(net.Resolver))
	ctx := context.Background()
	t.Run("Addr", func(t *testing.T) {
		names, err := resolver.LookupAddr(ctx, "8.8.8.8")
		if names == nil || err != nil {
			t.Fatal("unexpected result")
		}
	})
	t.Run("CNAME", func(t *testing.T) {
		name, err := resolver.LookupCNAME(ctx, "google.com")
		if name == "" || err != nil {
			t.Fatal("unexpected result")
		}
	})
	t.Run("MX", func(t *testing.T) {
		records, err := resolver.LookupMX(ctx, "google.com")
		if records == nil || err != nil {
			t.Fatal("unexpected result")
		}
	})
	t.Run("NS", func(t *testing.T) {
		records, err := resolver.LookupNS(ctx, "google.com")
		if records == nil || err != nil {
			t.Fatal("unexpected result")
		}
	})
}

type fakeresolverbogon struct {
	*net.Resolver
	err   error
	reply []string
}

func (c *fakeresolverbogon) LookupHost(
	ctx context.Context, hostname string,
) ([]string, error) {
	return c.reply, c.err
}
