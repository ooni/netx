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
	"github.com/ooni/netx/internal/dnsclient/emittingdnsclient"
	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
)

var nextConnID int64

func getNextConnID() int64 {
	return atomic.AddInt64(&nextConnID, 1)
}

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	Beginning             time.Time
	DialContextDep        func(context.Context, string, string) (net.Conn, error)
	Handler               model.Handler
	LookupHost            func(context.Context, string) ([]string, error)
	TLSConfig             *tls.Config
	TLSHandshakeTimeout   time.Duration
	startTLSHandshakeHook func(net.Conn)
}

// NewDialer creates a new Dialer.
func NewDialer(beginning time.Time, handler model.Handler) *Dialer {
	return &Dialer{
		Beginning:      beginning,
		DialContextDep: (&net.Dialer{}).DialContext,
		Handler:        handler,
		LookupHost: emittingdnsclient.New(&net.Resolver{
			// This is equivalent to ConfigureDNS("system", "...")
			PreferGo: true,
		}).LookupHost,
		TLSConfig:             &tls.Config{},
		startTLSHandshakeHook: func(net.Conn) {},
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
	conn, _, _, err = d.flexibleDial(ctx, network, address, false)
	if err != nil {
		// This is necessary because we're converting from
		// *measurement.Conn to net.Conn.
		return nil, err
	}
	return net.Conn(conn), nil
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, address string) (net.Conn, error) {
	return d.DialTLSContext(context.Background(), network, address)
}

// DialTLSContext is like DialTLS but with context.
func (d *Dialer) DialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	conn, onlyhost, _, err := d.flexibleDial(ctx, network, address, false)
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

func (d *Dialer) flexibleDial(
	ctx context.Context, network, address string, requireIP bool,
) (conn *connx.MeasuringConn, onlyhost, onlyport string, err error) {
	onlyhost, onlyport, err = net.SplitHostPort(address)
	if err != nil {
		return
	}
	connid := getNextConnID()
	if net.ParseIP(onlyhost) != nil {
		conn, err = d.dialHostPort(ctx, network, onlyhost, onlyport, connid)
		return
	}
	if requireIP == true {
		err = errors.New("dialerapi: you passed me a domain name")
		return
	}
	var addrs []string
	addrs, err = d.LookupHost(tracing.WithInfo(ctx, &tracing.Info{
		Beginning: d.Beginning,
		ConnID:    connid,
		Handler:   d.Handler,
	}), onlyhost)
	if err != nil {
		return
	}
	for _, addr := range addrs {
		conn, err = d.dialHostPort(ctx, network, addr, onlyport, connid)
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

func (d *Dialer) dialHostPort(
	ctx context.Context, network, onlyhost, onlyport string, connid int64,
) (*connx.MeasuringConn, error) {
	if net.ParseIP(onlyhost) == nil {
		return nil, errors.New("dialerapi: you passed me a domain name")
	}
	address := net.JoinHostPort(onlyhost, onlyport)
	start := time.Now()
	conn, err := d.DialContextDep(ctx, network, address)
	stop := time.Now()
	d.Handler.OnMeasurement(model.Measurement{
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
		Handler:   d.Handler,
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
