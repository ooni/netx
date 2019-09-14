// Package httpx contains OONI's net/http extensions
package httpx

import (
	"net/http"
	"time"

	"github.com/bassosimone/netx"
	"github.com/bassosimone/netx/httpx/httptracex"
)

// Client is OONI's HTTP client.
type Client struct {
	// http.Client is the base structure.
	http.Client

	// Dialer controls how we dial network connections.
	Dialer *netx.MeasuringDialer

	// Transport is the HTTP transport we use.
	Transport *http.Transport

	// Tracer controls HTTP tracing.
	Tracer *httptracex.Tracer
}

// NewClient creates a new OONI HTTP client.
func NewClient() (c *Client) {
	c = new(Client)
	beginning := time.Now()
	c.Dialer = netx.NewMeasuringDialer(beginning)
	c.Transport = &http.Transport{
		Dial:        c.Dialer.Dial,
		DialContext: c.Dialer.DialContext,
	}
	c.Tracer = &httptracex.Tracer{
		EventsContainer: httptracex.EventsContainer{
			Beginning: beginning,
		},
		RoundTripper: c.Transport,
	}
	c.Client = http.Client{Transport: c.Tracer}
	return
}

// HTTPEvents returns the gathered HTTP events.
func (c *Client) HTTPEvents() []httptracex.Event {
	return c.Tracer.EventsContainer.Events
}

// NetEvents returns the gathered net events.
func (c *Client) NetEvents() []netx.TimingMeasurement {
	return c.Dialer.TimingMeasurements
}
