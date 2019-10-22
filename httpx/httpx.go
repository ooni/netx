// Package httpx contains OONI's net/http extensions. It defines the Client and
// the Transport replacements that we should use in OONI. They emit measurements
// collected at network and HTTP level using a specific handler.
package httpx

import (
	"net/http"
	"time"

	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/internal/dnsconf"
	"github.com/ooni/netx/internal/httptransport"
	"github.com/ooni/netx/internal/tlsconf"
	"github.com/ooni/netx/model"
)

// Transport performs measurements during HTTP round trips.
type Transport struct {
	dialer    *dialerapi.Dialer
	transport *httptransport.Transport
}

// NewTransport creates a new Transport. The beginning argument is
// the time to use as zero for computing the elapsed time.
func NewTransport(beginning time.Time, handler model.Handler) *Transport {
	t := new(Transport)
	t.dialer = dialerapi.NewDialer(beginning, handler)
	t.transport = httptransport.NewTransport(beginning, handler)
	// make sure HTTP uses our dialer
	t.transport.DialContext = t.dialer.DialContext
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
	return dnsconf.ConfigureDNS(t.dialer, network, address)
}

// SetCABundle internally calls netx.Dialer.SetCABundle and
// therefore it has the same caveats and limitations.
func (t *Transport) SetCABundle(path string) error {
	return tlsconf.SetCABundle(t.transport.TLSClientConfig, path)
}

// ForceSpecificSNI forces using a specific SNI.
func (t *Transport) ForceSpecificSNI(sni string) error {
	return tlsconf.ForceSpecificSNI(t.transport.TLSClientConfig, sni)
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

// ConfigureDNS internally calls netx.Dialer.ConfigureDNS and
// therefore it has the same caveats and limitations.
func (c *Client) ConfigureDNS(network, address string) error {
	return c.Transport.ConfigureDNS(network, address)
}

// SetCABundle internally calls netx.Dialer.SetCABundle and
// therefore it has the same caveats and limitations.
func (c *Client) SetCABundle(path string) error {
	return c.Transport.SetCABundle(path)
}

// ForceSpecificSNI forces using a specific SNI.
func (c *Client) ForceSpecificSNI(sni string) error {
	return c.Transport.ForceSpecificSNI(sni)
}
