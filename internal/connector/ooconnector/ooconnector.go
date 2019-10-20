// Package ooconnector contains OONI's connector
package ooconnector

import (
	"context"
	"errors"
	"net"
)

// Connector is OONI's connector
type Connector struct{}

// New returns a new OONI connector
func New() *Connector {
	return new(Connector)
}

// DialContext creates a new connection.
func (c *Connector) DialContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	if h, _, e := net.SplitHostPort(address); e != nil || net.ParseIP(h) == nil {
		return nil, errors.New("ooconnector: didn't pass me a <ip>:<port>")
	}
	return (&net.Dialer{}).DialContext(ctx, network, address)
}
