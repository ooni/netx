// Package dialerbase contains the base dialer functionality. We connect
// to a remote endpoint, but we don't support DNS.
package dialerbase

import (
	"context"
	"net"
	"time"

	"github.com/ooni/netx/internal/connid"
	"github.com/ooni/netx/internal/dialer/connx"
	"github.com/ooni/netx/model"
)

// Dialer is a net.Dialer that is only able to connect to
// remote TCP/UDP endpoints. DNS is not supported.
type Dialer struct {
	dialer    model.Dialer
	beginning time.Time
	handler   model.Handler
	dialID    int64
}

// New creates a new dialer
func New(
	beginning time.Time,
	handler model.Handler,
	dialer model.Dialer,
	dialID int64,
) *Dialer {
	return &Dialer{
		dialer:    dialer,
		beginning: beginning,
		handler:   handler,
		dialID:    dialID,
	}
}

// Dial creates a TCP or UDP connection. See net.Dial docs.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialContext dials a new connection with context.
func (d *Dialer) DialContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	start := time.Now()
	conn, err := d.dialer.DialContext(ctx, network, address)
	stop := time.Now()
	connID := safeConnID(network, conn)
	d.handler.OnMeasurement(model.Measurement{
		Connect: &model.ConnectEvent{
			ConnID:        connID,
			DialID:        d.dialID,
			Duration:      stop.Sub(start),
			Error:         err,
			Network:       network,
			RemoteAddress: safeRemoteAddress(conn),
			Time:          stop.Sub(d.beginning),
		},
	})
	if err != nil {
		return nil, err
	}
	return &connx.MeasuringConn{
		Conn:      conn,
		Beginning: d.beginning,
		Handler:   d.handler,
		ID:        connID,
	}, nil
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

func safeConnID(network string, conn net.Conn) int64 {
	return connid.Compute(network, safeLocalAddress(conn))
}
