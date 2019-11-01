// Package tlsdialer contains the TLS dialer
package tlsdialer

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/ooni/netx/internal/dialer/connx"
	"github.com/ooni/netx/internal/errwrapper"
	"github.com/ooni/netx/model"
)

// TLSDialer is the TLS dialer
type TLSDialer struct {
	ConnectTimeout      time.Duration // default: 30 second
	TLSHandshakeTimeout time.Duration // default: 10 second
	config              *tls.Config
	dialer              model.Dialer
	setDeadline         func(net.Conn, time.Time) error
}

// New creates a new TLS dialer
func New(dialer model.Dialer, config *tls.Config) *TLSDialer {
	return &TLSDialer{
		ConnectTimeout:      30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		config:              config,
		dialer:              dialer,
		setDeadline: func(conn net.Conn, t time.Time) error {
			return conn.SetDeadline(t)
		},
	}
}

// DialTLS dials a new TLS connection
func (d *TLSDialer) DialTLS(network, address string) (net.Conn, error) {
	ctx := context.Background()
	return d.DialTLSContext(ctx, network, address)
}

// DialTLSContext is like DialTLS, but with context
func (d *TLSDialer) DialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, d.ConnectTimeout)
	defer cancel()
	conn, err := d.dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	config := d.config.Clone() // avoid polluting original config
	if config.ServerName == "" {
		config.ServerName = host
	}
	err = d.setDeadline(conn, time.Now().Add(d.TLSHandshakeTimeout))
	if err != nil {
		conn.Close()
		return nil, err
	}
	tlsconn := tls.Client(conn, config)
	var connID int64
	if mconn, ok := conn.(*connx.MeasuringConn); ok {
		connID = mconn.ID
	}
	root := model.ContextMeasurementRootOrDefault(ctx)
	// Implementation note: when DialTLS is not set, the code in
	// net/http will perform the handshake. Otherwise, if DialTLS
	// is set, we will end up here. This code is still used when
	// performing non-HTTP TLS-enabled dial operations.
	root.Handler.OnMeasurement(model.Measurement{
		TLSHandshakeStart: &model.TLSHandshakeStartEvent{
			ConnID:                 connID,
			DurationSinceBeginning: time.Now().Sub(root.Beginning),
		},
	})
	err = tlsconn.Handshake()
	err = errwrapper.SafeErrWrapperBuilder{
		ConnID:    connID,
		Error:     err,
		Operation: "tls_handshake",
	}.MaybeBuild()
	root.Handler.OnMeasurement(model.Measurement{
		TLSHandshakeDone: &model.TLSHandshakeDoneEvent{
			ConnID:                 connID,
			ConnectionState:        model.NewTLSConnectionState(tlsconn.ConnectionState()),
			Error:                  err,
			DurationSinceBeginning: time.Now().Sub(root.Beginning),
		},
	})
	conn.SetDeadline(time.Time{}) // clear deadline
	if err != nil {
		conn.Close()
		return nil, err
	}
	return tlsconn, err
}
