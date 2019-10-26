package tracetripper

import (
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"testing"
)

func TestIntegration(t *testing.T) {
	client := &http.Client{
		Transport: New(http.DefaultTransport),
	}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	client.CloseIdleConnections()
}

func TestIntegrationFailure(t *testing.T) {
	client := &http.Client{
		Transport: New(http.DefaultTransport),
	}
	// This fails the request because we attempt to speak cleartext HTTP with
	// a server that instead is expecting TLS.
	resp, err := client.Get("http://www.google.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resp != nil {
		t.Fatal("expected a nil response here")
	}
	client.CloseIdleConnections()
}

func TestIntegrationWithClientTrace(t *testing.T) {
	client := &http.Client{
		Transport: New(http.DefaultTransport),
	}
	req, err := http.NewRequest("GET", "https://www.kernel.org/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(
		httptrace.WithClientTrace(req.Context(), new(httptrace.ClientTrace)),
	)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected a good response here")
	}
	resp.Body.Close()
	client.CloseIdleConnections()
}
