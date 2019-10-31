package ooniresolver

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/netx/internal/resolver/dnstransport/dnsovertcp"
	"github.com/ooni/netx/internal/resolver/dnstransport/dnsoverudp"
	"github.com/ooni/netx/model"
)

func newtransport() model.DNSRoundTripper {
	return dnsovertcp.NewTransportTCP(&net.Dialer{}, "dns.quad9.net:53")
}

func TestGettingTransport(t *testing.T) {
	transport := newtransport()
	client := New(transport)
	if transport != client.Transport() {
		t.Fatal("the transport is not correctly set")
	}
}

func TestLookupAddr(t *testing.T) {
	client := New(newtransport())
	names, err := client.LookupAddr(context.Background(), "8.8.8.8")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if names != nil {
		t.Fatal("expected nil result here")
	}
}

func TestLookupCNAME(t *testing.T) {
	client := New(newtransport())
	cname, err := client.LookupCNAME(context.Background(), "www.ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if cname != "" {
		t.Fatal("expected empty result here")
	}
}

func TestLookupHostWithRetry(t *testing.T) {
	// Because there is no server there, if there is no DNS injection
	// then we are going to see several timeouts.
	client := New(dnsoverudp.NewTransport(
		&net.Dialer{}, "www.example.com:53",
	))
	addrs, err := client.LookupHost(context.Background(), "www.google.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err.Error() != "context deadline exceeded" {
		t.Fatal("not the error we expected")
	}
	if client.ntimeouts <= 0 {
		t.Fatal("no timeouts?")
	}
	if addrs != nil {
		t.Fatal("expected nil addr here")
	}
}

type faketransport struct{}

func (t *faketransport) RoundTrip(
	ctx context.Context, query []byte,
) (reply []byte, err error) {
	return nil, errors.New("mocked error")
}

func TestLookupHostWithNonTimeoutError(t *testing.T) {
	client := New(&faketransport{})
	addrs, err := client.LookupHost(context.Background(), "www.google.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	// Not a typo! Check for equality to make sure that we are
	// in the case where no timeout was returned but something else.
	if err.Error() == "context deadline exceeded" {
		t.Fatal("not the error we expected")
	}
	if client.ntimeouts != 0 {
		t.Fatal("we saw a timeout?")
	}
	if addrs != nil {
		t.Fatal("expected nil addr here")
	}
}

func TestLookupHost(t *testing.T) {
	client := New(newtransport())
	addrs, err := client.LookupHost(context.Background(), "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if addrs == nil {
		t.Fatal("expected non-nil result here")
	}
}

func TestLookupNonexistent(t *testing.T) {
	client := New(newtransport())
	addrs, err := client.LookupHost(context.Background(), "nonexistent.ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if addrs != nil {
		t.Fatal("expected nil addr here")
	}
}

func TestLookupMX(t *testing.T) {
	client := New(newtransport())
	records, err := client.LookupMX(context.Background(), "ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if records != nil {
		t.Fatal("expected nil result here")
	}
}

func TestLookupNS(t *testing.T) {
	client := New(newtransport())
	records, err := client.LookupNS(context.Background(), "ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if records != nil {
		t.Fatal("expected nil result here")
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
