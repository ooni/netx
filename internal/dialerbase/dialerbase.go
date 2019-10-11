// Package dialerbase contains the base dialer functionality. We connect
// to a remote endpoint, but we don't support DNS.
package dialerbase

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/model"
)

// Dialer is a net.Dialer that is only able to connect to
// remote TCP/UDP endpoints. DNS is not supported.
type Dialer struct {
	net.Dialer
	Beginning time.Time
}

// NewDialer creates a new base dialer
func NewDialer(beginning time.Time) *Dialer {
	return &Dialer{
		Dialer:    net.Dialer{},
		Beginning: beginning,
	}
}

// DialHostPort is like net.DialContext but requires a separate host
// and port and returns a measurable net.Conn-like struct.
func (d *Dialer) DialHostPort(
	ctx context.Context, handler model.Handler,
	network, onlyhost, onlyport string, connid int64,
) (*connx.MeasuringConn, error) {
	if net.ParseIP(onlyhost) == nil {
		return nil, errors.New("dialerbase: you passed me a domain name")
	}
	address := net.JoinHostPort(onlyhost, onlyport)
	start := time.Now()
	conn, err := d.Dialer.DialContext(ctx, network, address)
	stop := time.Now()
	handler.OnMeasurement(model.Measurement{
		Connect: &model.ConnectEvent{
			ConnID:        connid,
			Duration:      stop.Sub(start),
			Error:         err,
			LocalAddress:  safeLocalAddress(conn),
			Network:       network,
			RemoteAddress: safeRemoteAddress(conn),
			Time:          stop.Sub(d.Beginning),
		},
	})
	if err != nil {
		return nil, err
	}
	return &connx.MeasuringConn{
		Conn:      conn,
		Beginning: d.Beginning,
		Handler:   handler,
		ID:        connid,
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
