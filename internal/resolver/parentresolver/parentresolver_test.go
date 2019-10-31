package parentresolver

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ooni/netx/internal/resolver/systemresolver"
	"github.com/ooni/netx/model"
)

func TestLookupAddr(t *testing.T) {
	client := New(new(net.Resolver))
	names, err := client.LookupAddr(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	if names == nil {
		t.Fatal("expected non-nil result here")
	}
}

func TestLookupCNAME(t *testing.T) {
	client := New(new(net.Resolver))
	cname, err := client.LookupCNAME(context.Background(), "www.ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	if cname == "" {
		t.Fatal("expected non-empty result here")
	}
}

type emitterchecker struct {
	gotResolveStart bool
	gotResolveDone  bool
	mu              sync.Mutex
}

func (h *emitterchecker) OnMeasurement(m model.Measurement) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m.ResolveStart != nil {
		h.gotResolveStart = true
	}
	if m.ResolveDone != nil {
		h.gotResolveDone = true
	}
}

func TestLookupHost(t *testing.T) {
	client := New(systemresolver.New(new(net.Resolver)))
	handler := new(emitterchecker)
	ctx := model.WithMeasurementRoot(
		context.Background(), &model.MeasurementRoot{
			Beginning: time.Now(),
			Handler:   handler,
		})
	addrs, err := client.LookupHost(ctx, "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		t.Log(addr)
	}
	handler.mu.Lock()
	defer handler.mu.Unlock()
	if handler.gotResolveStart == false {
		t.Fatal("did not see resolve start event")
	}
	if handler.gotResolveDone == false {
		t.Fatal("did not see resolve done event")
	}
}

func TestLookupHostBogon(t *testing.T) {
	client := New(systemresolver.New(new(net.Resolver)))
	handler := new(emitterchecker)
	ctx := model.WithMeasurementRoot(
		context.Background(), &model.MeasurementRoot{
			Beginning: time.Now(),
			Handler:   handler,
		})
	addrs, err := client.LookupHost(ctx, "localhost")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err.Error() != "dns_bogon_error" {
		t.Fatal("not the error that we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil addr here")
	}
	root := model.ContextMeasurementRoot(ctx)
	if root.X.Scoreboard.DNSBogonInfo == nil {
		t.Fatal("no bogon info added to scoreboard")
	}
}

func TestLookupMX(t *testing.T) {
	client := New(new(net.Resolver))
	records, err := client.LookupMX(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	if records == nil {
		t.Fatal("expected non-nil result here")
	}
}

func TestLookupNS(t *testing.T) {
	client := New(new(net.Resolver))
	records, err := client.LookupNS(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	if records == nil {
		t.Fatal("expected non-nil result here")
	}
}
