// Package emittingconnector contains a connector emitting events
package emittingconnector

import (
	"context"
	"net"
	"time"

	"github.com/ooni/netx/internal/connector"
	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
)

// Connector is a connector emitting events
type Connector struct {
	connector connector.Model
}

// New returns a new emitting connector
func New(connector connector.Model) *Connector {
	return &Connector{connector: connector}
}

// DialContext creates a new connection.
func (c *Connector) DialContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	start := time.Now()
	conn, err := c.connector.DialContext(ctx, network, address)
	stop := time.Now()
	if info := tracing.ContextInfo(ctx); info != nil {
		info.Handler.OnMeasurement(model.Measurement{
			Connect: &model.ConnectEvent{
				ConnID:        info.ConnID,
				Duration:      stop.Sub(start),
				Error:         err,
				LocalAddress:  safeLocalAddress(conn),
				Network:       network,
				RemoteAddress: safeRemoteAddress(conn),
				Time:          stop.Sub(info.Beginning),
			},
		})
		if conn != nil {
			conn = &connx.MeasuringConn{
				Beginning: info.Beginning,
				Conn:      conn,
				Handler:   info.Handler,
				ID:        info.ConnID,
			}
		}
	}
	return conn, err
}

func safeLocalAddress(conn net.Conn) (s string) {
	if conn != nil && conn.LocalAddr() != nil {
		s = conn.LocalAddr().String()
	}
	return
}

func safeRemoteAddress(conn net.Conn) (s string) {
	if conn != nil && conn.RemoteAddr() != nil {
		s = conn.RemoteAddr().String()
	}
	return
}
