package emitterresolver

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
	addrs, err := client.LookupAddr(context.Background(), "130.192.91.211")
	if err == nil {
		t.Fatal("expected an error here")
	}
	for _, addr := range addrs {
		t.Log(addr)
	}
}

func TestLookupCNAME(t *testing.T) {
	client := New(new(net.Resolver))
	addrs, err := client.LookupCNAME(context.Background(), "www.ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		t.Log(addr)
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

func TestLookupMX(t *testing.T) {
	client := New(new(net.Resolver))
	addrs, err := client.LookupMX(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		t.Log(addr)
	}
}

func TestLookupNS(t *testing.T) {
	client := New(new(net.Resolver))
	addrs, err := client.LookupNS(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		t.Log(addr)
	}
}
