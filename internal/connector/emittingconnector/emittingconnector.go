// Package emittingconnector contains a connector emitting events
package emittingconnector

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/internal/connector"
	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
)

var connID int64

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
	cid := atomic.AddInt64(&connID, 1)
	start := time.Now()
	conn, err := c.connector.DialContext(ctx, network, address)
	stop := time.Now()
	if info := tracing.ContextInfo(ctx); info != nil {
		info.ConnID = cid
		info.Handler.OnMeasurement(model.Measurement{
			Connect: &model.ConnectEvent{
				SyscallEvent: model.SyscallEvent{
					BaseEvent:   info.BaseEvent(),
					BlockedTime: stop.Sub(start),
				},
				Error:         err,
				Network:       network,
				RemoteAddress: safeRemoteAddress(conn),
			},
		})
		if conn != nil {
			conn = connx.NewMeasuringConn(conn, info)
		}
	}
	return conn, err
}

func safeRemoteAddress(conn net.Conn) (s string) {
	if conn != nil && conn.RemoteAddr() != nil {
		s = conn.RemoteAddr().String()
	}
	return
}
