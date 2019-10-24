// Package dialer contains the dialer's API. The dialer defined
// in here implements basic DNS, but that is overridable.
package dialer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/internal/dialer/dialerbase"
	"github.com/ooni/netx/model"
)

var nextDialID, nextConnID int64

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	Beginning             time.Time
	Handler               model.Handler
	Resolver              model.DNSResolver
	StartTLSHandshakeHook func(net.Conn)
	TLSConfig             *tls.Config
	TLSHandshakeTimeout   time.Duration
}

// NewDialer creates a new Dialer.
func NewDialer(
	beginning time.Time, handler model.Handler,
) (d *Dialer) {
	return &Dialer{
		Beginning:             beginning,
		Handler:               handler,
		Resolver:              new(net.Resolver),
		TLSConfig:             &tls.Config{},
		StartTLSHandshakeHook: func(net.Conn) {},
	}
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
	conn, _, _, _, err = d.DialContextEx(ctx, network, address, false)
	if err != nil {
		// This is necessary because we're converting from
		// *measurement.Conn to net.Conn.
		return nil, err
	}
	return conn, nil
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, address string) (net.Conn, error) {
	ctx := context.Background()
	return d.DialTLSContext(ctx, network, address)
}

// DialTLSContext is like DialTLS, but with context
func (d *Dialer) DialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	conn, onlyhost, _, connID, err := d.DialContextEx(ctx, network, address, false)
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
	tc, err := d.tlsHandshake(config, timeout, conn, connID)
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
) (conn net.Conn, onlyhost, onlyport string, connID int64, err error) {
	onlyhost, onlyport, err = net.SplitHostPort(address)
	if err != nil {
		return
	}
	dialID := atomic.AddInt64(&nextDialID, 1)
	connID = atomic.AddInt64(&nextConnID, 1)
	if net.ParseIP(onlyhost) != nil {
		dialer := dialerbase.New(
			d.Beginning, d.Handler, new(net.Dialer), dialID, connID,
		)
		conn, err = dialer.DialContext(ctx, network, address)
		return
	}
	if requireIP == true {
		err = errors.New("dialer: you passed me a domain name")
		return
	}
	start := time.Now()
	var addrs []string
	addrs, err = d.Resolver.LookupHost(ctx, onlyhost)
	stop := time.Now()
	d.Handler.OnMeasurement(model.Measurement{
		Resolve: &model.ResolveEvent{
			Addresses: addrs,
			DialID:    dialID,
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
		dialer := dialerbase.New(
			d.Beginning, d.Handler, new(net.Dialer), dialID, connID,
		)
		target := net.JoinHostPort(addr, onlyport)
		conn, err = dialer.DialContext(ctx, network, target)
		if err == nil {
			return
		}
		connID = atomic.AddInt64(&nextConnID, 1)
	}
	err = &net.OpError{
		Op:  "dial",
		Net: network,
		Err: errors.New("all connect attempts failed"),
	}
	return
}

func (d *Dialer) clonedTLSConfig() *tls.Config {
	return d.TLSConfig.Clone()
}

func (d *Dialer) tlsHandshake(
	config *tls.Config, timeout time.Duration, conn net.Conn, connID int64,
) (*tls.Conn, error) {
	d.StartTLSHandshakeHook(conn)
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
			ConnID:   connID,
			Time:     stop.Sub(d.Beginning),
		},
	})
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

func simplifyCerts(in []*x509.Certificate) (out []model.X509Certificate) {
	for _, cert := range in {
		out = append(out, model.X509Certificate{
			Data: cert.Raw,
		})
	}
	return
}

// SetCABundle configures the dialer to use a specific CA bundle.
func (d *Dialer) SetCABundle(path string) error {
	cert, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(cert)
	d.TLSConfig.RootCAs = pool
	return nil
}

// ForceSpecificSNI forces using a specific SNI.
func (d *Dialer) ForceSpecificSNI(sni string) error {
	d.TLSConfig.ServerName = sni
	return nil
}
