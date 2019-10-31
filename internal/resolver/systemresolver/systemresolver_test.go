package systemresolver

import (
	"context"
	"net"
	"testing"

	"github.com/ooni/netx/model"
)

type queryableTransport interface {
	Network() string
	Address() string
}

type queryableResolver interface {
	Transport() model.DNSRoundTripper
}

func TestCanQuery(t *testing.T) {
	var client model.DNSResolver = New(new(net.Resolver))
	transport := client.(queryableResolver).Transport()
	reply, err := transport.RoundTrip(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err.Error() != "not implemented" {
		t.Fatal("not the error we expected")
	}
	if reply != nil {
		t.Fatal("expected nil reply here")
	}
	queryableTransport := transport.(queryableTransport)
	if queryableTransport.Address() != "" {
		t.Fatal("invalid address")
	}
	if queryableTransport.Network() != "system" {
		t.Fatal("invalid network")
	}
}

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

func TestLookupHost(t *testing.T) {
	client := New(new(net.Resolver))
	addrs, err := client.LookupHost(context.Background(), "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		t.Log(addr)
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
