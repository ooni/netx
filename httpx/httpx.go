// Package httpx contains OONI's net/http extensions. It defines the Client and
// the Transport replacements that we should use in OONI. They emit measurements
// collected at network and HTTP level on a specific channel.
package httpx

import (
	"net/http"
	"time"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dnsconf"
	"github.com/bassosimone/netx/internal/httptransport"
	"github.com/bassosimone/netx/model"
)

// Transport performs measurements during HTTP round trips.
type Transport struct {
	dialer    *dialerapi.Dialer
	transport *httptransport.Transport
}

// NewTransport creates a new Transport. The beginning argument is
// the time to use as zero for computing the elapsed time. The ch
// channel is where we'll emit Measurements.
func NewTransport(beginning time.Time, handler model.Handler) *Transport {
	t := new(Transport)
	t.dialer = dialerapi.NewDialer(beginning, handler)
	t.transport = httptransport.NewTransport(beginning, handler)
	// make sure we use an http2 ready TLS config
	t.dialer.TLSConfig = t.transport.TLSClientConfig
	// make sure HTTP uses our dialer
	t.transport.Dial = t.dialer.Dial
	t.transport.DialContext = t.dialer.DialContext
	t.transport.DialTLS = t.dialer.DialTLS
	return t
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.transport.RoundTrip(req)
}

// CloseIdleConnections closes any connections which were previously connected
// from previous requests but are now sitting idle in a "keep-alive" state. It
// does not interrupt any connections currently in use.
func (t *Transport) CloseIdleConnections() {
	t.transport.CloseIdleConnections()
}

// ConfigureDNS is exactly like netx.Dialer.ConfigureDNS.
func (t *Transport) ConfigureDNS(network, address string) error {
	return dnsconf.Do(t.dialer, network, address)
}

// Client is a replacement for http.Client.
type Client struct {
	// HTTPClient is the underlying client. Pass this client to existing code
	// that expects an *http.HTTPClient. For this reason we can't embed it.
	HTTPClient *http.Client

	// Transport is the transport configured by NewClient to be used
	// by the HTTPClient field.
	Transport *Transport
}

// NewClient creates a new client instance.
func NewClient(handler model.Handler) *Client {
	transport := NewTransport(time.Now(), handler)
	return &Client{
		HTTPClient: &http.Client{
			Transport: transport,
		},
		Transport: transport,
	}
}

// ConfigureDNS is exactly like netx.Dialer.ConfigureDNS.
func (c *Client) ConfigureDNS(network, address string) error {
	return c.Transport.ConfigureDNS(network, address)
}
