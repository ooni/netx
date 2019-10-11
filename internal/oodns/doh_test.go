package oodns

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/netx/dnsx"
)

func TestIntegrationDoHSuccess(t *testing.T) {
	transport := NewTransportDoH(
		http.DefaultClient, "https://cloudflare-dns.com/dns-query",
	)
	err := threeRounds(transport)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationNewRequestFailure(t *testing.T) {
	transport := NewTransportDoH(
		http.DefaultClient, "\t", // invalid URL
	)
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationClientDoFailure(t *testing.T) {
	transport := &dohTransport{
		clientDo: func(*http.Request) (*http.Response, error) {
			return nil, errors.New("mocked error")
		},
		url: "https://cloudflare-dns.com/dns-query",
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationHTTPFailure(t *testing.T) {
	transport := &dohTransport{
		clientDo: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 500,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil
		},
		url: "https://cloudflare-dns.com/dns-query",
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationMissingHeader(t *testing.T) {
	transport := &dohTransport{
		clientDo: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil
		},
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func threeRounds(transport dnsx.RoundTripper) error {
	err := roundTrip(transport, "ooni.io.")
	if err != nil {
		return err
	}
	err = roundTrip(transport, "slashdot.org.")
	if err != nil {
		return err
	}
	err = roundTrip(transport, "kernel.org.")
	if err != nil {
		return err
	}
	return nil
}

func roundTrip(transport dnsx.RoundTripper, domain string) error {
	query := new(dns.Msg)
	query.SetQuestion(domain, dns.TypeA)
	data, err := query.Pack()
	if err != nil {
		return err
	}
	data, err = transport.RoundTrip(data)
	if err != nil {
		return err
	}
	return query.Unpack(data)
}
