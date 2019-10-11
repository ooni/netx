package httptransport_test

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/httptransport"
	"github.com/ooni/netx/internal/tracing"
)

func TestIntegration(t *testing.T) {
	client := &http.Client{
		Transport: httptransport.NewTransport(time.Now(), handlers.NoHandler),
	}
	req, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	saver := tracing.NewSaver()
	req = req.WithContext(tracing.WithHandler(
		req.Context(), saver,
	))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(saver.Measurements) <= 1 {
		t.Fatal("No measurement was saved")
	}
}

func TestIntegrationFailure(t *testing.T) {
	client := &http.Client{
		Transport: httptransport.NewTransport(time.Now(), handlers.NoHandler),
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
}
