package httptransport

import (
	"net/http"
	"net/http/httptrace"
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

func TestIntegrationPanicHTTPTrace(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic here")
		}
	}()
	transport := newtransport()
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest("GET", "http://www.kernel.org/", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := req.Context()
	ctx = httptrace.WithClientTrace(ctx, new(httptrace.ClientTrace))
	req = req.WithContext(ctx)
	_, err = client.Do(req)
}

func newtransport() *Transport {
	return New(
		time.Now(),
		handlers.NoHandler,
		http.DefaultTransport,
		true,
	)
}
