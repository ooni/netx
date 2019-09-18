// Package dialerapi contains the dialer's API. The dialer defined
// in here implements basic DNS, but that is overridable.
package dialerapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/bassosimone/netx/internal/connx"
	"github.com/bassosimone/netx/internal/dialerbase"
	"github.com/bassosimone/netx/model"
)

var nextConnID int64

// NextConnID returns the next connection ID.
func NextConnID() int64 {
	return atomic.AddInt64(&nextConnID, 1)
}

type lookupHostFunc func(context.Context, string) ([]string, error)

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	dialerbase.Dialer
	Handler             model.Handler
	LookupHost          lookupHostFunc
	TLSConfig           *tls.Config
	TLSHandshakeTimeout time.Duration
}

// NewDialer creates a new Dialer.
func NewDialer(beginning time.Time, handler model.Handler) (d *Dialer) {
	d = &Dialer{
		Dialer: dialerbase.Dialer{
			Beginning: beginning,
			Dialer:    net.Dialer{},
			Handler:   handler,
		},
		Handler: handler,
	}
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			conn, _, _, err := d.DialContextEx(ctx, network, address, false)
			if err != nil {
				return nil, err
			}
			// convince Go this is really a net.PacketConn
			return &connx.DNSMeasuringConn{MeasuringConn: *conn}, nil
		},
	}
	d.LookupHost = r.LookupHost
	return
}

// Dial creates a TCP or UDP connection. See net.Dial docs.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialContext is like Dial but the context allows to interrupt a
// pending connection attempt at any time.
func (d *Dialer) DialContext(
	ctx context.Context, network, address string,
) (conn net.Conn, err error) {
	conn, _, _, err = d.DialContextEx(ctx, network, address, false)
	if err != nil {
		// This is necessary because we're converting from
		// *measurement.Conn to net.Conn.
		return nil, err
	}
	return net.Conn(conn), nil
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, address string) (net.Conn, error) {
	ctx := context.Background()
	conn, onlyhost, _, err := d.DialContextEx(ctx, network, address, false)
	if err != nil {
		return nil, err
	}
	config := d.clonedTLSConfig()
	if config.ServerName == "" {
		config.ServerName = onlyhost
	}
	timeout := d.TLSHandshakeTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	tc, err := d.tlsHandshake(config, timeout, conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	// Note that we cannot wrap `tc` because the HTTP code assumes
	// a `*tls.Conn` when implementing ALPN.
	return tc, nil
}

// DialContextEx is an extended DialContext where we may also
// optionally prevent processing of domain names.
func (d *Dialer) DialContextEx(
	ctx context.Context, network, address string, requireIP bool,
) (conn *connx.MeasuringConn, onlyhost, onlyport string, err error) {
	onlyhost, onlyport, err = net.SplitHostPort(address)
	if err != nil {
		return
	}
	connid := NextConnID()
	if net.ParseIP(onlyhost) != nil {
		conn, err = d.Dialer.DialHostPort(ctx, network, onlyhost, onlyport, connid)
		return
	}
	if requireIP == true {
		err = errors.New("dialerapi: you passed me a domain name")
		return
	}
	start := time.Now()
	var addrs []string
	addrs, err = d.LookupHost(ctx, onlyhost)
	stop := time.Now()
	d.Handler.OnMeasurement(model.Measurement{
		Resolve: &model.ResolveEvent{
			Addresses: addrs,
			ConnID:    connid,
			Duration:  stop.Sub(start),
			Error:     err,
			Hostname:  onlyhost,
			Time:      stop.Sub(d.Beginning),
		},
	})
	if err != nil {
		return
	}
	for _, addr := range addrs {
		conn, err = d.Dialer.DialHostPort(ctx, network, addr, onlyport, connid)
		if err == nil {
			return
		}
	}
	err = &net.OpError{
		Op:  "dial",
		Net: network,
		Err: errors.New("all connect attempts failed"),
	}
	return
}

func (d *Dialer) clonedTLSConfig() (config *tls.Config) {
	if d.TLSConfig != nil {
		config = d.TLSConfig.Clone()
	} else {
		config = &tls.Config{}
	}
	return
}

func (d *Dialer) tlsHandshake(
	config *tls.Config, timeout time.Duration, conn *connx.MeasuringConn,
) (tc *tls.Conn, err error) {
	tc = tls.Client(net.Conn(conn), config)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ech := make(chan error, 1)
	start := time.Now()
	go func() {
		ech <- tc.Handshake()
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-ech:
		// FALLTHROUGH
	}
	stop := time.Now()
	state := tc.ConnectionState()
	d.Handler.OnMeasurement(model.Measurement{
		TLSHandshake: &model.TLSHandshakeEvent{
			Config: model.TLSConfig{
				NextProtos: config.NextProtos,
				ServerName: config.ServerName,
			},
			ConnectionState: model.TLSConnectionState{
				CipherSuite:                state.CipherSuite,
				NegotiatedProtocol:         state.NegotiatedProtocol,
				NegotiatedProtocolIsMutual: state.NegotiatedProtocolIsMutual,
				PeerCertificates:           simplifyCerts(state.PeerCertificates),
				Version:                    state.Version,
			},
			Duration: stop.Sub(start),
			Error:    err,
			ConnID:   conn.ID,
			Time:     stop.Sub(conn.Beginning),
		},
	})
	if err != nil {
		tc.Close()
		tc = nil
	}
	return
}

func simplifyCerts(in []*x509.Certificate) (out []model.X509Certificate) {
	for _, cert := range in {
		out = append(out, model.X509Certificate{
			Data: cert.Raw,
		})
	}
	return
}
