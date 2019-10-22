package httptransport_test

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/httptransport"
	"github.com/ooni/netx/internal/tracing"
)

func TestIntegrationSuccess(t *testing.T) {
	testurl(t, "https://www.google.com", false, false)
}

func TestIntegrationSuccessWithHandler(t *testing.T) {
	testurl(t, "https://www.google.com", true, false)
}

func TestIntegrationFailure(t *testing.T) {
	// This fails the request because we attempt to speak cleartext HTTP with
	// a server that instead is expecting TLS.
	testurl(t, "http://www.google.com:443", false, true)
}

func TestIntegrationFailureWithHandler(t *testing.T) {
	// This fails the request because we attempt to speak cleartext HTTP with
	// a server that instead is expecting TLS.
	testurl(t, "http://www.google.com:443", true, true)
}

func testurl(t *testing.T, URL string, enableTracing, expectFailure bool) {
	client := &http.Client{Transport: httptransport.NewTransport()}
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if enableTracing {
		req = req.WithContext(tracing.WithInfo(req.Context(), &tracing.Info{
			Handler: handlers.NoHandler,
		}))
	}
	resp, err := client.Do(req)
	if expectFailure {
		if err == nil {
			t.Fatal("expected error")
		}
		if resp != nil {
			t.Fatal("expected nil response")
		}
	} else {
		if err != nil {
			t.Fatal(err)
		}
		if resp == nil {
			t.Fatal("nil response")
		}
		if _, err := ioutil.ReadAll(resp.Body); err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
}
