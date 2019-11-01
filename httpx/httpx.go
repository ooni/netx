// Package httpx contains OONI's net/http extensions. It defines the Client and
// the Transport replacements that we should use in OONI. They emit measurements
// collected at network and HTTP level using a specific handler.
package httpx

import (
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/netx/internal"
	"github.com/ooni/netx/model"
)

// Transport performs measurements during HTTP round trips.
type Transport struct {
	dialer    *internal.Dialer
	transport *internal.HTTPTransport
}

func newTransport(
	beginning time.Time, handler model.Handler,
	proxyFunc func(*http.Request) (*url.URL, error),
) *Transport {
	t := new(Transport)
	t.dialer = internal.NewDialer(beginning, handler)
	t.transport = internal.NewHTTPTransport(
		beginning,
		handler,
		t.dialer,
		false, // DisableKeepAlives
		proxyFunc,
	)
	return t
}

// NewTransport creates a new Transport. The beginning argument is
// the time to use as zero for computing the elapsed time.
func NewTransport(beginning time.Time, handler model.Handler) *Transport {
	return newTransport(beginning, handler, http.ProxyFromEnvironment)
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
	return t.dialer.ConfigureDNS(network, address)
}

// SetResolver is exactly like netx.Dialer.SetResolver.
func (t *Transport) SetResolver(r model.DNSResolver) {
	t.dialer.SetResolver(r)
}

// SetCABundle internally calls netx.Dialer.SetCABundle and
// therefore it has the same caveats and limitations.
func (t *Transport) SetCABundle(path string) error {
	return t.dialer.SetCABundle(path)
}

// ForceSpecificSNI forces using a specific SNI.
func (t *Transport) ForceSpecificSNI(sni string) error {
	return t.dialer.ForceSpecificSNI(sni)
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

func newClient(
	handler model.Handler,
	proxyFunc func(*http.Request) (*url.URL, error),
) *Client {
	transport := newTransport(time.Now(), handler, proxyFunc)
	return &Client{
		HTTPClient: &http.Client{
			Transport: transport,
		},
		Transport: transport,
	}
}

// NewClient creates a new client instance.
func NewClient(handler model.Handler) *Client {
	return newClient(handler, http.ProxyFromEnvironment)
}

// NewClientWithoutProxy creates a client without any
// configured proxy attached to it. This is suitable
// for measurements where you don't want a proxy to be
// in the middle and alter the measurements.
func NewClientWithoutProxy(handler model.Handler) *Client {
	return newClient(handler, nil)
}

// ConfigureDNS internally calls netx.Dialer.ConfigureDNS and
// therefore it has the same caveats and limitations.
func (c *Client) ConfigureDNS(network, address string) error {
	return c.Transport.ConfigureDNS(network, address)
}

// SetResolver internally calls netx.Dialer.SetResolver
func (c *Client) SetResolver(r model.DNSResolver) {
	c.Transport.SetResolver(r)
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
