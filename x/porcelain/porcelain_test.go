package porcelain

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ooni/netx/handlers"
)

func TestIntegration(t *testing.T) {
	body := strings.NewReader("antani")
	req, err := NewHTTPRequest("POST", "http://www.x.org", body)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "POST" {
		t.Fatal("unexpected method")
	}
	if req.URL.Scheme != "http" {
		t.Fatal("unexpected scheme")
	}
	if req.URL.Host != "www.x.org" {
		t.Fatal("unexpected host")
	}
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte("antani")) {
		t.Fatal("unexpected body")
	}
	root := RequestMeasurementRoot(req)
	if root == nil {
		t.Fatal("unexpected nil root")
	}
}

func TestGetWithRedirects(t *testing.T) {
	client := NewHTTPXClient()
	measurements, err := Get(
		handlers.NoHandler,
		client,
		"https://httpbin.org/redirect/4",
		"ooniprobe-netx/0.1.0",
	)
	if err != nil {
		t.Fatal(err)
	}
	if measurements == nil {
		t.Fatal("nil measurements")
	}
	if len(measurements.Resolves) < 1 {
		t.Fatal("no resolves?!")
	}
	if len(measurements.Connects) < 1 {
		t.Fatal("no connects?!")
	}
	if len(measurements.Requests) < 1 {
		t.Fatal("no requests?!")
	}
	if measurements.Scoreboard == nil {
		t.Fatal("no scoreboard?!")
	}
}

func TestGetWithInvalidURL(t *testing.T) {
	client := NewHTTPXClient()
	measurements, err := Get(
		handlers.NoHandler,
		client,
		"\t", // invalid URL
		"ooniprobe-netx/0.1.0",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if measurements != nil {
		t.Fatal("expected nil measurements")
	}
}

func TestTLSConnectNormal(t *testing.T) {
	measurements, err := TLSConnect(
		handlers.NoHandler,
		"example.com:443",
		"example.com",
	)
	if err != nil {
		t.Fatal(err)
	}
	if measurements == nil {
		t.Fatal("expected measurements")
	}
	if measurements.Error != err {
		t.Fatal("errors mismatch")
	}
}

func TestTLSConnectWrongSNI(t *testing.T) {
	measurements, err := TLSConnect(
		handlers.NoHandler,
		"example.com:443",
		"ooni.io",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err.Error() != "ssl_invalid_hostname" {
		t.Fatal("not the error we expected")
	}
	if measurements == nil {
		t.Fatal("expected measurements")
	}
	if measurements.Error != err {
		t.Fatal("errors mismatch")
	}
}

func TestTLSConnectWrongDomain(t *testing.T) {
	measurements, err := TLSConnect(
		handlers.NoHandler,
		"antani.local:443",
		"ooni.io",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err.Error() != "dns_nxdomain_error" {
		t.Fatal("not the error we expected")
	}
	if measurements == nil {
		t.Fatal("expected measurements")
	}
	if measurements.Error != err {
		t.Fatal("errors mismatch")
	}
}
