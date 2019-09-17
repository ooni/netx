// Package dialerapi contains the dialer's API. The dialer defined
// in here implements basic DNS, but that is overridable.
package dialerapi

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/bassosimone/netx/internal/connx"
	"github.com/bassosimone/netx/internal/dialerbase"
	"github.com/bassosimone/netx/internal/tlsx"
	"github.com/bassosimone/netx/model"
)

var nextConnID int64

type lookupHostFunc func(context.Context, string) ([]string, error)

func lookupHost(ctx context.Context, address string) ([]string, error) {
	return (&net.Resolver{}).LookupHost(ctx, address)
}

// Dialer defines the dialer API. We implement the most basic form
// of DNS, but more advanced resolutions are possible.
type Dialer struct {
	dialerbase.Dialer
	C                   chan model.Measurement
	LookupHost          lookupHostFunc
	TLSConfig           *tls.Config
	TLSHandshakeTimeout time.Duration
}

// NewDialer creates a new Dialer.
func NewDialer(beginning time.Time, ch chan model.Measurement) *Dialer {
	return &Dialer{
		Dialer: dialerbase.Dialer{
			Beginning: beginning,
			C:         ch,
			Dialer:    net.Dialer{},
		},
		C:          ch,
		LookupHost: lookupHost,
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
	conn, _, _, err = d.DialContextEx(ctx, network, address, false)
	return
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, address string) (net.Conn, error) {
	return d.DialTLSWithSNI(network, address, "")
}

// DialTLSWithSNI is like DialTLS, but using a different SNI. If the SNI
// is empty, this function is equivalent to DialTLS.
func (d *Dialer) DialTLSWithSNI(network, address, SNI string) (net.Conn, error) {
	ctx := context.Background()
	conn, onlyhost, _, err := d.DialContextEx(ctx, network, address, false)
	if err != nil {
		return nil, err
	}
	config := d.clonedTLSConfig()
	if SNI == "" {
		SNI = onlyhost
	}
	config.ServerName = SNI
	timeout := d.TLSHandshakeTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	tc, err := tlsx.Handshake(ctx, config, timeout, conn, d.C)
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
	connid := atomic.AddInt64(&nextConnID, 1)
	if net.ParseIP(onlyhost) != nil {
		conn, err = d.Dialer.DialHostPort(ctx, network, onlyhost, onlyport, connid)
		return
	}
	if requireIP == true {
		err = errors.New("dialerapi: you passed me a domain name")
	}
	start := time.Now()
	var addrs []string
	addrs, err = d.LookupHost(ctx, onlyhost)
	stop := time.Now()
	d.safesend(model.Measurement{
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

func (d *Dialer) safesend(m model.Measurement) {
	if d.C != nil {
		d.C <- m
	}
}
