// Package emittingdnsclient is a DNS client emitting events
package emittingdnsclient

import (
	"context"
	"net"
	"time"

	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
)

// Client is a DNS client that emits events
type Client struct {
	client model.DNSClient
}

// New creates a new emitting DNS client
func New(client model.DNSClient) *Client {
	return &Client{client: client}
}

// LookupAddr returns the name of the provided IP address
func (c *Client) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	return c.client.LookupAddr(ctx, addr)
}

// LookupCNAME returns the canonical name of a host
func (c *Client) LookupCNAME(ctx context.Context, host string) (string, error) {
	return c.client.LookupCNAME(ctx, host)
}

// LookupHost returns the IP addresses of a host
func (c *Client) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	start := time.Now()
	var addrs []string
	addrs, err := c.client.LookupHost(ctx, hostname)
	stop := time.Now()
	if info := tracing.ContextInfo(ctx); info != nil {
		info.Handler.OnMeasurement(model.Measurement{
			Resolve: &model.ResolveEvent{
				Addresses: addrs,
				ConnID:    info.ConnID,
				Duration:  stop.Sub(start),
				Error:     err,
				Hostname:  hostname,
				Time:      stop.Sub(info.Beginning),
			},
		})
	}
	return addrs, err
}

// LookupMX returns the MX records of a specific name
func (c *Client) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return c.client.LookupMX(ctx, name)
}

// LookupNS returns the NS records of a specific name
func (c *Client) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	return c.client.LookupNS(ctx, name)
}
