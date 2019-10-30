package tracetripper

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"sync"
	"testing"
	"time"

	"github.com/ooni/netx/model"
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

type redirectHandler struct {
	roundTrips []*model.HTTPRoundTripDoneEvent
	mu         sync.Mutex
}

func (h *redirectHandler) OnMeasurement(m model.Measurement) {
	if m.HTTPRoundTripDone != nil {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.roundTrips = append(h.roundTrips, m.HTTPRoundTripDone)
	}
}

func TestIntegrationRedirect(t *testing.T) {
	client := &http.Client{
		Transport: New(http.DefaultTransport),
	}
	req, err := http.NewRequest("GET", "https://google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := &redirectHandler{}
	ctx := model.WithMeasurementRoot(
		context.Background(),
		&model.MeasurementRoot{
			Beginning: time.Now(),
			Handler:   handler,
		},
	)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	client.CloseIdleConnections()
	handler.mu.Lock()
	defer handler.mu.Unlock()
	var wrong bool
	for _, ev := range handler.roundTrips {
		if ev.StatusCode >= 301 && ev.StatusCode <= 308 {
			if len(ev.RedirectBody) > 0 {
				wrong = false
			}
		}
	}
	if wrong {
		t.Fatal("seen a redirect without a body where it shouldn't")
	}
}

func TestIntegrationRedirectReadAllFailure(t *testing.T) {
	transport := New(http.DefaultTransport)
	transport.readAll = func(r io.Reader) ([]byte, error) {
		return nil, io.EOF
	}
	client := &http.Client{Transport: transport}
	resp, err := client.Get("https://google.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
	if transport.readAllErrs <= 0 {
		t.Fatal("not the error we expected")
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
