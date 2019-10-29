package ooniresolver

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/netx/internal/resolver/dnstransport/dnsovertcp"
	"github.com/ooni/netx/model"
)

func newtransport() model.DNSRoundTripper {
	return dnsovertcp.NewTransportTCP(&net.Dialer{}, "dns.quad9.net:53")
}

func TestLookupAddr(t *testing.T) {
	client := New(newtransport())
	addrs, err := client.LookupAddr(context.Background(), "130.192.91.211")
	if err == nil {
		t.Fatal("expected an error here")
	}
	for _, addr := range addrs {
		t.Log(addr)
	}
}

func TestLookupCNAME(t *testing.T) {
	client := New(newtransport())
	addrs, err := client.LookupCNAME(context.Background(), "www.ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	for _, addr := range addrs {
		t.Log(addr)
	}
}

func TestLookupHost(t *testing.T) {
	client := New(newtransport())
	addrs, err := client.LookupHost(context.Background(), "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		t.Log(addr)
	}
}

func TestLookupNonexistent(t *testing.T) {
	client := New(newtransport())
	addrs, err := client.LookupHost(context.Background(), "nonexistent.ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if addrs != nil {
		t.Fatal("expeced nil addr here")
	}
}

func TestLookupMX(t *testing.T) {
	client := New(newtransport())
	addrs, err := client.LookupMX(context.Background(), "ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	for _, addr := range addrs {
		t.Log(addr)
	}
}

func TestLookupNS(t *testing.T) {
	client := New(newtransport())
	addrs, err := client.LookupNS(context.Background(), "ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	for _, addr := range addrs {
		t.Log(addr)
	}
}

func TestRoundTripExPackFailure(t *testing.T) {
	client := New(newtransport())
	_, err := client.mockableRoundTrip(
		context.Background(), nil,
		func(msg *dns.Msg) ([]byte, error) {
			return nil, errors.New("mocked error")
		},
		func(t model.DNSRoundTripper, query []byte) (reply []byte, err error) {
			return nil, nil
		},
		func(msg *dns.Msg, data []byte) (err error) {
			return nil
		},
	)
	if err == nil {
		t.Fatal("expeced an error here")
	}
}

func TestRoundTripExRoundTripFailure(t *testing.T) {
	client := New(newtransport())
	_, err := client.mockableRoundTrip(
		context.Background(), nil,
		func(msg *dns.Msg) ([]byte, error) {
			return nil, nil
		},
		func(t model.DNSRoundTripper, query []byte) (reply []byte, err error) {
			return nil, errors.New("mocked error")
		},
		func(msg *dns.Msg, data []byte) (err error) {
			return nil
		},
	)
	if err == nil {
		t.Fatal("expeced an error here")
	}
}

func TestRoundTripExUnpackFailure(t *testing.T) {
	client := New(newtransport())
	_, err := client.mockableRoundTrip(
		context.Background(), nil,
		func(msg *dns.Msg) ([]byte, error) {
			return nil, nil
		},
		func(t model.DNSRoundTripper, query []byte) (reply []byte, err error) {
			return nil, nil
		},
		func(msg *dns.Msg, data []byte) (err error) {
			return errors.New("mocked error")
		},
	)
	if err == nil {
		t.Fatal("expeced an error here")
	}
}

func TestLookupHostResultNoName(t *testing.T) {
	addrs, err := lookupHostResult(nil, nil, nil)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs")
	}
}

func TestLookupHostResultAAAAError(t *testing.T) {
	addrs, err := lookupHostResult(nil, nil, errors.New("mocked error"))
	if err == nil {
		t.Fatal("expected an error here")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs")
	}
}
