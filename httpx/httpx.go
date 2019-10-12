// Package httpx contains OONI's net/http extensions. It defines the Client and
// the Transport replacements that we should use in OONI. They emit measurements
// collected at network and HTTP level using a specific handler.
package httpx

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/dialercontext"
	"github.com/ooni/netx/internal/httptransport"
	"github.com/ooni/netx/internal/oodns"
	"github.com/ooni/netx/internal/tlsx"
	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
)

// Transport performs measurements during HTTP round trips.
type Transport struct {
	dialer    *dialercontext.Dialer
	transport *httptransport.Transport
}

// NewTransport creates a new Transport. The beginning argument is
// the time to use as zero for computing the elapsed time.
func NewTransport(beginning time.Time, handler model.Handler) *Transport {
	t := new(Transport)
	t.dialer = dialercontext.NewDialer(beginning)
	t.transport = httptransport.NewTransport(beginning, handler)
	//
	// Implementation note: we use a reduced-complexity dialer that only
	// exposes DialContext because we are using a context for storing the
	// per-request handler, and DialTLS does not take a context.
	//
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

// ConfigureDNS behaves exactly like netx.Dialer.ConfigureDNS.
func (t *Transport) ConfigureDNS(network, address string) error {
	if network == "system" {
		t.dialer.LookupHost = (&net.Resolver{PreferGo: false}).LookupHost
		return nil
	}
	if network == "netgo" {
		t.dialer.LookupHost = (&net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				conn, _, _, err := t.dialer.DialContextEx(
					ctx, tracing.ContextHandler(ctx), network, address, false,
				)
				// convince Go this is really a net.PacketConn
				return &connx.DNSMeasuringConn{MeasuringConn: *conn}, err
			},
		}).LookupHost
		return nil
	}
	if network == "doh" {
		child := NewTransport(t.transport.Beginning, t.transport.Handler)
		resolver := oodns.NewClient(oodns.NewTransportDoH(&http.Client{
			Transport: child,
		}, address))
		t.dialer.LookupHost = resolver.LookupHost
		return nil
	}
	if network == "udp" {
		resolver := oodns.NewClient(oodns.NewTransportUDP(
			address, dialercontext.NewDialer(t.transport.Beginning).DialContext,
		))
		t.dialer.LookupHost = resolver.LookupHost
		return nil
	}
	if network == "tcp" {
		resolver := oodns.NewClient(oodns.NewTransportTCP(
			address, dialercontext.NewDialer(t.transport.Beginning).DialContext,
		))
		t.dialer.LookupHost = resolver.LookupHost
		return nil
	}
	// TODO(bassosimone): here we should re-enable all DNS transports.
	if network == "dot" {
		return nil // laying!
	}
	return errors.New("not implemented")
}

// SetCABundle internally calls netx.Dialer.SetCABundle and
// therefore it has the same caveats and limitations.
func (t *Transport) SetCABundle(path string) error {
	pool, err := tlsx.ReadCABundle(path)
	if err != nil {
		return err
	}
	t.transport.TLSClientConfig.RootCAs = pool
	return nil
}

// ForceSpecificSNI forces using a specific SNI.
func (t *Transport) ForceSpecificSNI(sni string) error {
	t.transport.TLSClientConfig.ServerName = sni
	return nil
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
