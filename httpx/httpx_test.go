package httpx_test

import (
	"io/ioutil"
	"testing"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/httpx"
)

func TestIntegration(t *testing.T) {
	client := httpx.NewClient(handlers.NoHandler)
	defer client.Transport.CloseIdleConnections()
	resp, err := client.HTTPClient.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationSystemDNS(t *testing.T) {
	client := httpx.NewClient(handlers.NoHandler)
	defer client.Transport.CloseIdleConnections()
	err := client.ConfigureDNS("system", "")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.HTTPClient.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationDoH(t *testing.T) {
	client := httpx.NewClient(handlers.NoHandler)
	defer client.Transport.CloseIdleConnections()
	err := client.ConfigureDNS("doh", "https://cloudflare-dns.com/dns-query")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.HTTPClient.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationDoT(t *testing.T) {
	client := httpx.NewClient(handlers.NoHandler)
	defer client.Transport.CloseIdleConnections()
	err := client.ConfigureDNS("dot", "1.1.1.1:853")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.HTTPClient.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationTCP(t *testing.T) {
	client := httpx.NewClient(handlers.NoHandler)
	defer client.Transport.CloseIdleConnections()
	err := client.ConfigureDNS("tcp", "1.1.1.1:53")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.HTTPClient.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationUDP(t *testing.T) {
	client := httpx.NewClient(handlers.NoHandler)
	defer client.Transport.CloseIdleConnections()
	err := client.ConfigureDNS("udp", "1.1.1.1:53")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.HTTPClient.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationNetgo(t *testing.T) {
	client := httpx.NewClient(handlers.NoHandler)
	defer client.Transport.CloseIdleConnections()
	err := client.ConfigureDNS("netgo", "")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.HTTPClient.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationInvalidDNS(t *testing.T) {
	client := httpx.NewClient(handlers.NoHandler)
	defer client.Transport.CloseIdleConnections()
	err := client.ConfigureDNS("invalid", "https://cloudflare-dns.com/dns-query")
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestSetCABundleGood(t *testing.T) {
	client := httpx.NewClient(handlers.NoHandler)
	err := client.SetCABundle("../testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetCABundleNonexistent(t *testing.T) {
	client := httpx.NewClient(handlers.NoHandler)
	err := client.SetCABundle("../testdata/cacert-nonexistent.pem")
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestForceSpecificSNI(t *testing.T) {
	client := httpx.NewClient(handlers.NoHandler)
	err := client.ForceSpecificSNI("www.facebook.com")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.HTTPClient.Get("https://www.google.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	// TODO(bassosimone): how to unwrap the error in Go < 1.13? Anyway we are
	// already testing we're getting the right error in netx_test.go.
	t.Log(err)
	if resp != nil {
		t.Fatal("expected a nil response here")
	}
}
