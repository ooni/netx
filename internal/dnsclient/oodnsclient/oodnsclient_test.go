package oodnsclient

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/netx/dnsx"
	"github.com/ooni/netx/internal/dnstransport/dnsovertcp"
)

func newtransport() dnsx.RoundTripper {
	return dnsovertcp.NewTransport(
		func(ctx context.Context, network, address string) (net.Conn, error) {
			return tls.Dial(network, address, nil)
		},
		"dns.quad9.net:853",
	)
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

func TestDoRoundTripPackFailure(t *testing.T) {
	client := New(newtransport())
	_, err := client.doRoundTrip(
		context.Background(), nil,
		func(msg *dns.Msg) ([]byte, error) {
			return nil, errors.New("mocked error")
		},
		func(t dnsx.RoundTripper, query []byte) (reply []byte, err error) {
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

func TestDoRoundTripRoundTripFailure(t *testing.T) {
	client := New(newtransport())
	_, err := client.doRoundTrip(
		context.Background(), nil,
		func(msg *dns.Msg) ([]byte, error) {
			return nil, nil
		},
		func(t dnsx.RoundTripper, query []byte) (reply []byte, err error) {
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

func TestDoRoundTripUnpackFailure(t *testing.T) {
	client := New(newtransport())
	_, err := client.doRoundTrip(
		context.Background(), nil,
		func(msg *dns.Msg) ([]byte, error) {
			return nil, nil
		},
		func(t dnsx.RoundTripper, query []byte) (reply []byte, err error) {
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
