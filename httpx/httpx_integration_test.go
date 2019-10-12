package httpx_test

// This file contains longer and/or more complex tests that go
// beyond merely checking the basic functionality.

import (
	"io/ioutil"
	"net/http"
	"sync"
	"testing"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/httpx"
	"github.com/ooni/netx/httpx/httptracex"
	"github.com/ooni/netx/internal/tracing"
)

// TestParallelismAndDataCollection is here to show that we can
// run round-trips in parallel and still collect data that makes
// sense (i.e. we can easily link net and http data).
func TestParallelismAndDataCollection(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	client := httpx.NewClient(handlers.NoHandler).HTTPClient
	results := map[string]*tracing.Saver{
		"http://www.x.org/":           nil,
		"http://www.slashdot.org/":    nil,
		"http://www.youtube.com/":     nil,
		"http://ooni.torproject.org/": nil,
	}
	wg := new(sync.WaitGroup)
	wg.Add(len(results))
	for key, _ := range results {
		saver := tracing.NewSaver()
		results[key] = saver
		req, err := http.NewRequest("GET", key, nil)
		if err != nil {
			t.Fatal(err)
		}
		req = httptracex.RequestWithHandler(req, saver)
		go func(wg *sync.WaitGroup, req *http.Request) {
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			wg.Done()
		}(wg, req)
	}
	wg.Wait()
	for _, value := range results {
		if len(value.Measurements) < 1 {
			t.Fatal("expected measurements here")
		}
		var (
			seenConnect   int
			seenHandshake int
			seenResolve   int
		)
		for _, m := range value.Measurements {
			if m.Resolve != nil {
				seenResolve++
			}
			if m.Connect != nil {
				seenConnect++
			}
			if m.TLSHandshake != nil {
				seenHandshake++
			}
		}
		if seenResolve < 1 {
			t.Fatal("did not see any resolve")
		}
		if seenConnect < 1 {
			t.Fatal("did not see any connect")
		}
		if seenHandshake < 1 {
			t.Fatal("did not see any TLS handshake")
		}
	}
}
