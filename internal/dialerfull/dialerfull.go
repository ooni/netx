// Package dialerfull contains a dialer that can do DialContext
// as well as DialTLS, thus implementing the full dialer API.
//
// This will eventually replace dialerapi.
package dialerfull

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/ooni/netx/internal/dialercontext"
	"github.com/ooni/netx/internal/tlsx"
	"github.com/ooni/netx/internal/tracing"
)

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	Beginning             time.Time
	TLSConfig             *tls.Config
	TLSHandshakeTimeout   time.Duration
	dialer                *dialercontext.Dialer
	startTLSHandshakeHook func(net.Conn)
}

// NewDialer creates a new Dialer.
func NewDialer(beginning time.Time) (d *Dialer) {
	d = &Dialer{
		Beginning: beginning,
		TLSConfig: &tls.Config{},
		dialer: dialercontext.NewDialer(
			beginning,
		),
		startTLSHandshakeHook: func(net.Conn) {},
	}
	return
}

// DialContext is like net.Dial but the context allows to interrupt a
// pending connection attempt at any time.
func (d *Dialer) DialContext(
	ctx context.Context, network, address string,
) (conn net.Conn, err error) {
	return d.dialer.DialContext(ctx, network, address)
}

// DialTLSContext creates a TLS connection.
func (d *Dialer) DialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	// TODO(bassosimone): here we're basically ignoring the context
	hostname, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	conn, err := d.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	config := d.clonedTLSConfig()
	if config.ServerName == "" {
		config.ServerName = hostname
	}
	timeout := d.TLSHandshakeTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	tc, err := d.tlsHandshake(ctx, config, timeout, conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	// Note that we cannot wrap `tc` because the HTTP code assumes
	// a `*tls.Conn` when implementing ALPN.
	return tc, nil
}

func (d *Dialer) clonedTLSConfig() *tls.Config {
	return d.TLSConfig.Clone()
}

func (d *Dialer) tlsHandshake(
	ctx context.Context, config *tls.Config,
	timeout time.Duration, conn net.Conn,
) (*tls.Conn, error) {
	d.startTLSHandshakeHook(conn)
	err := conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		conn.Close()
		return nil, err
	}
	tc := tls.Client(net.Conn(conn), config)
	start := time.Now()
	err = tc.Handshake()
	stop := time.Now()
	state := tc.ConnectionState()
	// Join the dialer's handler with the handler that was possibly
	// passed as part of the current context and emit on both
	tlsx.EmitTLSHandshakeEvent(
		tracing.ContextHandler(ctx),
		state,
		stop.Sub(d.Beginning),
		stop.Sub(start),
		err,
		config,
	)
	if err != nil {
		tc.Close()
		return nil, err
	}
	// The following call fails if the connection is not connected
	// which should not be the case at this point. If the connection
	// has just been disconnected, we'll notice when doing I/O, so
	// it is fine to ignore the return value of SetDeadline.
	tc.SetDeadline(time.Time{})
	return tc, nil
}

// SetCABundle configures the dialer to use a specific CA bundle.
func (d *Dialer) SetCABundle(path string) error {
	pool, err := tlsx.ReadCABundle(path)
	if err != nil {
		return err
	}
	d.TLSConfig.RootCAs = pool
	return nil
}

// ForceSpecificSNI forces using a specific SNI.
func (d *Dialer) ForceSpecificSNI(sni string) error {
	d.TLSConfig.ServerName = sni
	return nil
}
