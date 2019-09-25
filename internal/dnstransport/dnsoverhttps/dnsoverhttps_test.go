package dnsoverhttps_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/dnstransport/dnsoverhttps"
)

func TestIntegrationSuccess(t *testing.T) {
	transport := dnsoverhttps.NewTransport(
		time.Now(), handlers.NoHandler,
		"https://cloudflare-dns.com/dns-query",
	)
	err := threeRounds(transport)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationNewRequestFailure(t *testing.T) {
	transport := dnsoverhttps.NewTransport(
		time.Now(), handlers.NoHandler,
		"\t", // invalid URL
	)
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationClientDoFailure(t *testing.T) {
	transport := dnsoverhttps.NewTransport(
		time.Now(), handlers.NoHandler,
		"https://cloudflare-dns.com/dns-query",
	)
	transport.ClientDo = func(*http.Request) (*http.Response, error) {
		return nil, errors.New("mocked error")
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationHTTPFailure(t *testing.T) {
	transport := dnsoverhttps.NewTransport(
		time.Now(), handlers.NoHandler,
		"https://cloudflare-dns.com/dns-query",
	)
	transport.ClientDo = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 500,
			Body:       ioutil.NopCloser(strings.NewReader("")),
		}, nil
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationMissingHeader(t *testing.T) {
	transport := dnsoverhttps.NewTransport(
		time.Now(), handlers.NoHandler,
		"https://cloudflare-dns.com/dns-query",
	)
	transport.ClientDo = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader("")),
		}, nil
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func threeRounds(transport *dnsoverhttps.Transport) error {
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

func roundTrip(transport *dnsoverhttps.Transport, domain string) error {
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
