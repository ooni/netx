package oodns

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/netx/dnsx"
)

func TestLookupAddr(t *testing.T) {
	client := NewClient(NewTransportDoH(
		http.DefaultClient, "https://cloudflare-dns.com/dns-query",
	))
	addrs, err := client.LookupAddr(context.Background(), "130.192.91.211")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs")
	}
}

func TestLookupCNAME(t *testing.T) {
	client := NewClient(NewTransportDoH(
		http.DefaultClient, "https://cloudflare-dns.com/dns-query",
	))
	addr, err := client.LookupCNAME(context.Background(), "www.ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if addr != "" {
		t.Fatal("expected empty string")
	}
}

func TestLookupHost(t *testing.T) {
	client := NewClient(NewTransportDoH(
		http.DefaultClient, "https://cloudflare-dns.com/dns-query",
	))
	addrs, err := client.LookupHost(context.Background(), "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if addrs == nil {
		t.Fatal("expected non nil addrs")
	}
}

func TestLookupNonexistent(t *testing.T) {
	client := NewClient(NewTransportDoH(
		http.DefaultClient, "https://cloudflare-dns.com/dns-query",
	))
	addrs, err := client.LookupHost(context.Background(), "nonexistent.ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if addrs != nil {
		t.Fatal("expeced nil addr here")
	}
}

func TestLookupMX(t *testing.T) {
	client := NewClient(NewTransportDoH(
		http.DefaultClient, "https://cloudflare-dns.com/dns-query",
	))
	entries, err := client.LookupMX(context.Background(), "ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if entries != nil {
		t.Fatal("expected nil entries")
	}
}

func TestLookupNS(t *testing.T) {
	client := NewClient(NewTransportDoH(
		http.DefaultClient, "https://cloudflare-dns.com/dns-query",
	))
	entries, err := client.LookupNS(context.Background(), "ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if entries != nil {
		t.Fatal("expected nil entries")
	}
}

func TestRoundTripExPackFailure(t *testing.T) {
	client := NewClient(NewTransportDoH(
		http.DefaultClient, "https://cloudflare-dns.com/dns-query",
	))
	_, err := client.roundTripEx(
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

func TestRoundTripExRoundTripFailure(t *testing.T) {
	client := NewClient(NewTransportDoH(
		http.DefaultClient, "https://cloudflare-dns.com/dns-query",
	))
	_, err := client.roundTripEx(
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

func TestRoundTripExUnpackFailure(t *testing.T) {
	client := NewClient(NewTransportDoH(
		http.DefaultClient, "https://cloudflare-dns.com/dns-query",
	))
	_, err := client.roundTripEx(
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
