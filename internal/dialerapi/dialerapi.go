// Package dialerapi contains the dialer's API. The dialer defined
// in here implements basic DNS, but that is overridable.
package dialerapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"sync/atomic"
	"time"

	"github.com/ooni/netx/internal/connx"
	"github.com/ooni/netx/internal/dialerbase"
	"github.com/ooni/netx/model"
)

var nextConnID int64

// NextConnID returns the next connection ID.
func NextConnID() int64 {
	return atomic.AddInt64(&nextConnID, 1)
}

// LookupHostFunc is the type of the function used to lookup
// the addresses of a specific host.
type LookupHostFunc func(context.Context, string) ([]string, error)

// DialHostPortFunc is the type of the function that is actually
// used to dial a connection to a specific host and port.
type DialHostPortFunc func(
	ctx context.Context, network, onlyhost, onlyport string, connid int64,
) (*connx.MeasuringConn, error)

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	Beginning             time.Time
	DialHostPort          DialHostPortFunc
	Handler               model.Handler
	LookupHost            LookupHostFunc
	StartTLSHandshakeHook func(net.Conn)
	TLSConfig             *tls.Config
	TLSHandshakeTimeout   time.Duration
	dialer                *dialerbase.Dialer
}

// NewDialer creates a new Dialer.
func NewDialer(beginning time.Time, handler model.Handler) (d *Dialer) {
	d = &Dialer{
		Beginning:             beginning,
		Handler:               handler,
		TLSConfig:             &tls.Config{},
		StartTLSHandshakeHook: func(net.Conn) {},
		dialer: dialerbase.NewDialer(
			beginning, handler,
		),
	}
	// This is equivalent to ConfigureDNS("system", "...")
	r := &net.Resolver{
		PreferGo: false,
	}
	d.LookupHost = r.LookupHost
	d.DialHostPort = d.dialer.DialHostPort
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
	return d.DialTLSContext(ctx, network, address)
}

// DialTLSContext is like DialTLS, but with context
func (d *Dialer) DialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
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
		conn, err = d.DialHostPort(ctx, network, onlyhost, onlyport, connid)
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
		conn, err = d.DialHostPort(ctx, network, addr, onlyport, connid)
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

func (d *Dialer) clonedTLSConfig() *tls.Config {
	return d.TLSConfig.Clone()
}

func (d *Dialer) tlsHandshake(
	config *tls.Config, timeout time.Duration, conn *connx.MeasuringConn,
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
			ConnID:   conn.ID,
			Time:     stop.Sub(conn.Beginning),
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
