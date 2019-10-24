package httptransport

import (
	"net/http"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
)

func TestIntegrationSuccess(t *testing.T) {
	transport := newtransport()
	client := &http.Client{Transport: transport}
	resp, err := client.Get("http://www.facebook.com/")
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("nil resp")
	}
	// if this WAIs, we should cover transport.CloseIdleConnections
	client.CloseIdleConnections()
}

func newtransport() *Transport {
	return New(
		time.Now(),
		handlers.NoHandler,
		http.DefaultTransport,
		true,
	)
}
