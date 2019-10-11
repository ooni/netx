// Package dnsoverhttps implements DNS over HTTPS.
//
// This package will eventually be replaced by code in oodns.
package dnsoverhttps

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/internal/httptransport"
	"github.com/ooni/netx/model"
)

// Transport is a DNS over HTTPS dnsx.RoundTripper.
//
// As a known bug, this implementation does not cache the domain
// name in the URL for reuse, but this should be easy to fix.
type Transport struct {
	// Client is the HTTP client to use.
	Client *http.Client

	// ClientDo allows to override the Client.Do behaviour. This is
	// initialized in NewTransport to call Client.Do.
	ClientDo func(req *http.Request) (*http.Response, error)

	// URL is the DoH server URL.
	URL string
}

// NewTransport creates a new Transport
func NewTransport(beginning time.Time, handler model.Handler, URL string) *Transport {
	dialer := dialerapi.NewDialer(beginning, handler)
	transport := httptransport.NewTransport(dialer.Beginning, dialer.Handler)
	// Logic to make sure we'll use the dialer in the new HTTP transport
	dialer.TLSConfig = transport.TLSClientConfig
	transport.Dial = dialer.Dial
	transport.DialContext = dialer.DialContext
	transport.DialTLS = dialer.DialTLS
	transport.MaxConnsPerHost = 1 // seems to be better for cloudflare DNS
	client := &http.Client{Transport: transport}
	return &Transport{
		Client:   client,
		ClientDo: client.Do,
		URL:      URL,
	}
}

// RoundTrip sends a request and receives a response.
func (t *Transport) RoundTrip(query []byte) ([]byte, error) {
	return t.RoundTripContext(context.Background(), query)
}

// RoundTripContext is like RoundTrip but with context.
func (t *Transport) RoundTripContext(
	ctx context.Context, query []byte,
) (reply []byte, err error) {
	req, err := http.NewRequest("POST", t.URL, bytes.NewReader(query))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/dns-message")
	var resp *http.Response
	resp, err = t.ClientDo(req.WithContext(ctx))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		// TODO(bassosimone): we should map the status code to a
		// proper Error in the DNS context.
		err = errors.New("doh: server returned error")
		return
	}
	if resp.Header.Get("content-type") != "application/dns-message" {
		err = errors.New("doh: invalid content-type")
		return
	}
	reply, err = ioutil.ReadAll(resp.Body)
	return
}
