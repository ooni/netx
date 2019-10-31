package porcelain

import (
	"context"
	"testing"
)

func TestIntegrationDNSLookupGood(t *testing.T) {
	ctx := context.Background()
	results, err := DNSLookup(ctx, DNSLookupConfig{
		Hostname: "ooni.io",
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error != nil {
		t.Fatal(err)
	}
	if len(results.Addresses) < 1 {
		t.Fatal("no addresses returned?!")
	}
}

func TestIntegrationDNSLookupUnknownDNS(t *testing.T) {
	ctx := context.Background()
	results, err := DNSLookup(ctx, DNSLookupConfig{
		Hostname:      "ooni.io",
		ServerNetwork: "antani",
	})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if results != nil {
		t.Fatal("expected nil results here")
	}
}

func TestIntegrationHTTPDoGood(t *testing.T) {
	ctx := context.Background()
	results, err := HTTPDo(ctx, HTTPDoConfig{
		URL: "http://ooni.io",
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error != nil {
		t.Fatal(err)
	}
	if results.StatusCode != 200 {
		t.Fatal("request failed?!")
	}
	if len(results.Headers) < 1 {
		t.Fatal("no headers?!")
	}
	if len(results.Body) < 1 {
		t.Fatal("no body?!")
	}
}

func TestIntegrationHTTPDoUnknownDNS(t *testing.T) {
	ctx := context.Background()
	results, err := HTTPDo(ctx, HTTPDoConfig{
		URL:              "http://ooni.io",
		DNSServerNetwork: "antani",
	})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if results != nil {
		t.Fatal("expected nil results here")
	}
}

func TestIntegrationHTTPDoRoundTripError(t *testing.T) {
	ctx := context.Background()
	results, err := HTTPDo(ctx, HTTPDoConfig{
		URL: "http://ooni.io:443", // 443 with http
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationHTTPDoBadURL(t *testing.T) {
	ctx := context.Background()
	results, err := HTTPDo(ctx, HTTPDoConfig{
		URL: "\t",
	})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if results != nil {
		t.Fatal("expected nil results here")
	}
}

func TestIntegrationTLSConnectGood(t *testing.T) {
	ctx := context.Background()
	results, err := TLSConnect(ctx, TLSConnectConfig{
		Address: "ooni.io:443",
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error != nil {
		t.Fatal(err)
	}
}

func TestIntegrationTLSConnectUnknownDNS(t *testing.T) {
	ctx := context.Background()
	results, err := TLSConnect(ctx, TLSConnectConfig{
		Address:          "ooni.io:443",
		DNSServerNetwork: "antani",
	})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if results != nil {
		t.Fatal("expected nil results here")
	}
}
