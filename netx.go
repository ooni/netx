// Package netx contains OONI's net extensions.
//
// This package provides a replacement for net.Dialer that can Dial,
// DialContext, and DialTLS. During its lifecycle this modified Dialer
// will observe network level events and collect Measurements.
//
// Each net.Conn created using this modified Dialer has an unique
// int64 network connection identifier, which is used to correlate
// network level events with, e.g., http level events. The identifier
// is assigned when Dial-ing the connection. Therefore, if multiple
// connection attempts are made to a specific endpoint (e.g. because
// it is down), you will see multiple ConnectOperation events that
// are using the same identifier. We never reuse IDs, but in theory the
// internal int64 counter we use could wrap around.
//
// Measurement is the data structure used by all the network
// level events we collect. The OperationID field of a specific
// Measurement identifies the measured operation, and determines
// what fields are meaningful.
//
// Use dialer.PopMeasurements() at any time to extract all the
// measurements collected so far.
package netx

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/bassosimone/netx/internal"
	"github.com/bassosimone/netx/logx"
)

// OperationID is the ID of a network-level operation.
type OperationID string

const (
	// CloseOperation is the ID of net.Conn.Close.
	CloseOperation = OperationID("close")

	// ConnectOperation is the ID of the operation where you connect
	// a BSD socket to a specified network endpoint.
	ConnectOperation = OperationID("connect")

	// ReadOperation is the ID of net.Conn.Read.
	ReadOperation = OperationID("read")

	// ResolveOperation is the ID of net.Resolver.LookupHost.
	ResolveOperation = OperationID("resolve")

	// TLSHandshakeOperation is the ID of tls.Conn.Handshake.
	TLSHandshakeOperation = OperationID("tlsHandshake")

	// WriteOperation is the ID of net.Conn.Write.
	WriteOperation = OperationID("write")
)

// Measurement describes a timing measurement. Some fields
// should be always present, others are optional. The optional
// fields are also marked with `json:",omitempty"`.
type Measurement struct {
	// Address is the address used by a ConnectOperation.
	Address string `json:",omitempty"`

	// Addresses contains the addresses returned by a ResolveOperation.
	Addresses []string `json:",omitempty"`

	// ConnID is the ID of the connection. Note that the ID is assigned
	// when we start dialing, therefore, all connection attempts part
	// of a single dial operation will have the same ID.
	ConnID int64

	// Data is the data transferred by {Read,Write}Operation. Note that this
	// field is currently only filled by DNS I/O operations.
	Data []byte `json:",omitempty"`

	// Duration is the operation's duration.
	Duration time.Duration

	// Error is the error that occurred, or nil.
	Error error

	// Hostname is the hostname passed to a ResolveOperation.
	Hostname string `json:",omitempty"`

	// Network is the network used by a ConnectOperation.
	Network string `json:",omitempty"`

	// NumBytes is the number of bytes transferred by {Read,Write}Operation.
	NumBytes int64 `json:",omitempty"`

	// OperationID is the operation's ID.
	OperationID OperationID

	// SNI is the SNI used by TLSHandshakeOperation.
	SNI string `json:",omitempty"`

	// StartTime is the time when the operation started relative to the
	// moment stored in Dialer.Beginning.
	StartTime time.Duration

	// TLSConnectionState is the TLS state of TLSHandshakeOperation.
	TLSConnectionState *tls.ConnectionState `json:",omitempty"`
}

// Dialer creates connections and collects Measurements.
type Dialer struct {
	// Beginning is the point in time considered as the beginning
	// of the measurements performed by this Dialer. This field is
	// initialized by the NewDialer constructor.
	Beginning time.Time

	// BytesRead counts the bytes read by all the connections created
	// using this specific dialer.
	BytesRead int64

	// BytesWritten counts the bytes written by all the connections
	// created using this specific dialer.
	BytesWritten int64

	// Dialer is the child Dialer. It is initialized by the NewDialer
	// constructor such that there is a reasonable timeout for
	// establishing TCP connections.
	Dialer *net.Dialer

	// EnableFullTiming controls whether to also measure the timing of
	// I/O operations. By default, we only measure the timing of the
	// connect and TLS handshake operations, and the I/O operations we
	// perform when doing DNS lookups. By setting this field to true,
	// you enable also timing all other Read and Write operations.
	EnableFullTiming bool

	// Logger is the interface used for logging. By default we use a
	// dummy logger that does nothing, but you may want logging.
	Logger logx.Logger

	// LookupHost is the function called to perform host lookups by this
	// dialer. By default uses the embedded Dialer's resolver. To implement
	// e.g. DoT or DoH, override this function.
	LookupHost func(ctx context.Context, host string) (addrs []string, err error)

	// TLSConfig is the configuration used by TLS. If this field is nil, we
	// will use a default TLS configuration.
	TLSConfig *tls.Config

	// TLSHandshakeTimeout is the timeout for TLS handshakes. If this field is
	// zero, or negative, we will use a default timeout.
	TLSHandshakeTimeout time.Duration

	connID       int64
	measurements []Measurement
	mutex        sync.Mutex
}

// NewDialer returns a new Dialer instance.
func NewDialer(beginning time.Time) (d *Dialer) {
	d = new(Dialer)
	d.Beginning = beginning
	d.LookupHost = func(ctx context.Context, host string) ([]string, error) {
		return d.Dialer.Resolver.LookupHost(ctx, host)
	}
	d.Dialer = &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial:     d.dialForDNS,
		},
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	d.Logger = &internal.NoLogger{}
	return
}

// PopMeasurements extracts the measurements collected by this dialer and
// returns them in a goroutine safe way.
func (d *Dialer) PopMeasurements() (measurements []Measurement) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	measurements = d.measurements
	d.measurements = nil
	return
}

func (d *Dialer) append(m Measurement) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.measurements = append(d.measurements, m)
}

type measurableConn struct {
	net.Conn
	dialer      *Dialer
	includeData bool
	sessID      int64
}

func (c *measurableConn) Read(b []byte) (n int, err error) {
	var start time.Time
	if c.dialer.EnableFullTiming {
		start = time.Now()
	}
	n, err = c.Conn.Read(b)
	if n > 0 {
		atomic.AddInt64(&c.dialer.BytesRead, int64(n))
	}
	if c.dialer.EnableFullTiming || c.includeData {
		m := Measurement{
			Duration:    time.Now().Sub(start),
			Error:       err,
			NumBytes:    int64(n),
			OperationID: ReadOperation,
			ConnID:      c.sessID,
			StartTime:   start.Sub(c.dialer.Beginning),
		}
		if c.includeData && n > 0 {
			m.Data = b[:n]
		}
		c.dialer.append(m)
		c.dialer.Logger.Debugf("(conn #%d) read %d bytes", c.sessID, n)
		if err != nil {
			c.dialer.Logger.Debugf("(conn #%d) %s", c.sessID, err.Error())
		}
	}
	return
}

func (c *measurableConn) Write(b []byte) (n int, err error) {
	var start time.Time
	if c.dialer.EnableFullTiming {
		start = time.Now()
	}
	n, err = c.Conn.Write(b)
	if n > 0 {
		atomic.AddInt64(&c.dialer.BytesWritten, int64(n))
	}
	if c.dialer.EnableFullTiming || c.includeData {
		m := Measurement{
			Duration:    time.Now().Sub(start),
			Error:       err,
			NumBytes:    int64(n),
			OperationID: WriteOperation,
			ConnID:      c.sessID,
			StartTime:   start.Sub(c.dialer.Beginning),
		}
		if c.includeData && n > 0 {
			m.Data = b[:n]
		}
		c.dialer.append(m)
		c.dialer.Logger.Debugf("(conn #%d) written %d bytes", c.sessID, n)
		if err != nil {
			c.dialer.Logger.Debugf("(conn #%d) %s", c.sessID, err.Error())
		}
	}
	return
}

func (c *measurableConn) Close() (err error) {
	start := time.Now()
	err = c.Conn.Close()
	c.dialer.append(Measurement{
		Duration:    time.Now().Sub(start),
		Error:       err,
		OperationID: CloseOperation,
		ConnID:      c.sessID,
		StartTime:   start.Sub(c.dialer.Beginning),
	})
	c.dialer.Logger.Debugf("(conn #%d) close", c.sessID)
	if err != nil {
		c.dialer.Logger.Debugf("(conn #%d) %s", c.sessID, err.Error())
	}
	return
}

// asPacketConn is required by Go's dnsclient, which behaves
// differently depending on the type of connection. Specifically
// the code casts to net.PacketConn to decide whether it needs
// to use TCP or UDP. So, measurableConn cannot just satisfy the
// interface of net.PacketConn. Rather, we need an adapter with
// which to wrap a measurableConn that is also a PacketConn.
type asPacketConn struct {
	measurableConn
}

func (c *asPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	// We don't need to measure this operation currently, so we have
	// not bothered with collecting the data.
	if c, ok := c.Conn.(net.PacketConn); ok {
		return c.ReadFrom(p)
	}
	err = net.Error(&net.OpError{ // should not happen
		Op:     "ReadFrom",
		Source: c.Conn.LocalAddr(),
		Addr:   c.Conn.RemoteAddr(),
		Err:    syscall.ENOTCONN,
	})
	return
}

func (c *asPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	// We don't need to measure this operation currently, so we have
	// not bothered with collecting the data.
	if c, ok := c.Conn.(net.PacketConn); ok {
		return c.WriteTo(p, addr)
	}
	err = net.Error(&net.OpError{ // should not happen
		Op:     "WriteTo",
		Source: c.Conn.LocalAddr(),
		Addr:   c.Conn.RemoteAddr(),
		Err:    syscall.ENOTCONN,
	})
	return
}

// GetConnID returns the conn's unique ID.
func GetConnID(conn net.Conn, id *int64) error {
	if c, ok := conn.(*measurableConn); ok {
		*id = c.sessID
		return nil
	}
	if c, ok := conn.(*asPacketConn); ok {
		*id = c.sessID
		return nil
	}
	if c, ok := conn.(*tlsConnWrapper); ok {
		*id = c.sessID
		return nil
	}
	return errors.New("netx: not a connection we know of")
}

// Dial creates a TCP or UDP connection.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialContext is like Dial but the context allows to interrupt a
// pending connection attempt.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.newDialerEx(network, address, false).dial(ctx)
}

func (d *Dialer) dialForDNS(ctx context.Context, network, address string) (net.Conn, error) {
	return d.newDialerEx(network, address, true).dial(ctx)
}

func (d *Dialer) newDialerEx(network, address string, includeData bool) *dialerEx {
	return &dialerEx{
		address:     address,
		dialer:      d,
		id:          atomic.AddInt64(&d.connID, 1),
		includeData: includeData,
		network:     network,
	}
}

type dialerEx struct {
	dialer      *Dialer // dialer to use
	network     string  // network to connect to
	address     string  // address for connect
	includeData bool    // include data?
	id          int64   // conn ID
}

type errManyConnectFailed struct {
	Errors []error
}

func (errManyConnectFailed) Error() string {
	return "netx.DialContext: cannot connect any of the specified addresses"
}
func (errManyConnectFailed) Timeout() bool {
	return false
}
func (errManyConnectFailed) Temporary() bool {
	return false
}

func (de *dialerEx) dial(ctx context.Context) (net.Conn, error) {
	de.dialer.Logger.Debugf(
		"(conn #%d) dial %s/%s", de.id, de.address, de.network,
	)
	host, port, err := net.SplitHostPort(de.address)
	if err != nil {
		return nil, err
	}
	if net.ParseIP(host) != nil {
		return de.connect(ctx, host, port)
	}
	addrs, err := de.lookup(ctx, host)
	if err != nil {
		return nil, err
	}
	var multierr errManyConnectFailed
	for _, addr := range addrs {
		conn, err := de.connect(ctx, addr, port)
		if err == nil {
			return conn, nil
		}
		multierr.Errors = append(multierr.Errors, err)
	}
	if len(multierr.Errors) == 1 {
		return nil, multierr.Errors[0] // Unwrap when we have just one error
	}
	return nil, net.Error(multierr)
}

func (de *dialerEx) lookup(ctx context.Context, host string) ([]string, error) {
	de.dialer.Logger.Debugf("(conn #%d) lookup %s", de.id, host)
	start := time.Now()
	addrs, err := de.dialer.LookupHost(ctx, host)
	de.dialer.append(Measurement{
		Addresses:   addrs,
		Duration:    time.Now().Sub(start),
		Error:       err,
		Hostname:    host,
		OperationID: ResolveOperation,
		ConnID:      de.id,
		StartTime:   start.Sub(de.dialer.Beginning),
	})
	if err != nil {
		de.dialer.Logger.Debug(err.Error())
	}
	return addrs, err
}

func (de *dialerEx) connect(ctx context.Context, addr, port string) (net.Conn, error) {
	if net.ParseIP(addr) == nil {
		return nil, errors.New("dialContextAddrPort: expected an address")
	}
	// Assumption: dial using an IP address boils down to connect
	addrport := net.JoinHostPort(addr, port)
	de.dialer.Logger.Debugf("(conn #%d) connect %s/%s", de.id, addrport, de.network)
	start := time.Now()
	conn, err := de.dialer.Dialer.DialContext(ctx, de.network, addrport)
	de.dialer.append(Measurement{
		Address:     addrport,
		Duration:    time.Now().Sub(start),
		Error:       err,
		Network:     de.network,
		OperationID: ConnectOperation,
		ConnID:      de.id,
		StartTime:   start.Sub(de.dialer.Beginning),
	})
	if err != nil {
		de.dialer.Logger.Debug(err.Error())
		return nil, err
	}
	return de.wrapConn(conn), nil
}

func (de *dialerEx) wrapConn(conn net.Conn) net.Conn {
	if _, ok := conn.(net.PacketConn); ok {
		// When a connection is a PacketConn, make sure we return a
		// structure that matches the PacketConn interface.
		return &asPacketConn{
			measurableConn: measurableConn{
				Conn:        conn,
				dialer:      de.dialer,
				includeData: de.includeData,
				sessID:      de.id,
			},
		}
	}
	return &measurableConn{
		Conn:        conn,
		dialer:      de.dialer,
		includeData: de.includeData,
		sessID:      de.id,
	}
}

type tlsConnWrapper struct {
	net.Conn
	sessID int64
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, addr string) (conn net.Conn, err error) {
	defer func() {
		if err != nil && conn != nil {
			conn.Close()
			conn = nil
		}
	}()
	hostname, _, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	conn, err = d.Dial(network, addr)
	if err != nil {
		return
	}
	var connid int64
	err = GetConnID(conn, &connid)
	if err != nil {
		return
	}
	var config *tls.Config
	if d.TLSConfig != nil {
		config = d.TLSConfig.Clone()
	} else {
		config = &tls.Config{}
	}
	config.ServerName = hostname
	handshakeTimeout := d.TLSHandshakeTimeout
	if handshakeTimeout <= 0 {
		handshakeTimeout = 10 * time.Second
	}
	tlsconn := tls.Client(conn, config)
	ctx, cancel := context.WithTimeout(context.Background(), handshakeTimeout)
	defer cancel()
	ch := make(chan error)
	d.Logger.Debugf("(conn #%d) tls: start handshake", connid)
	start := time.Now()
	go func() {
		ch <- tlsconn.Handshake()
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-ch:
		// FALLTHROUGH
	}
	d.Logger.Debugf("(conn #%d) tls: handshake done", connid)
	connstate := tlsconn.ConnectionState()
	d.append(Measurement{
		Duration:           time.Now().Sub(start),
		Error:              err,
		OperationID:        TLSHandshakeOperation,
		SNI:                config.ServerName,
		ConnID:             connid,
		StartTime:          start.Sub(d.Beginning),
		TLSConnectionState: &connstate,
	})
	if err != nil {
		d.Logger.Debugf("(conn #%d) tls: %s", connid, err.Error())
		return
	}
	d.Logger.Debugf("(conn #%d) SSL connection using %s / %s",
		connid, internal.TLSVersionAsString(connstate.Version),
		internal.TLSCipherSuiteAsString(connstate.CipherSuite),
	)
	d.Logger.Debugf("(conn #%d) ALPN negotiated protocol: %s",
		connid, internal.TLSNegotiatedProtocol(connstate.NegotiatedProtocol),
	)
	for idx, cert := range connstate.PeerCertificates {
		d.Logger.Debugf("(conn #%d) %d: Subject: %s", connid, idx, cert.Subject.String())
		d.Logger.Debugf("(conn #%d) %d: NotBefore: %s", connid, idx, cert.NotBefore.String())
		d.Logger.Debugf("(conn #%d) %d: NotAfter: %s", connid, idx, cert.NotAfter.String())
		d.Logger.Debugf("(conn #%d) %d: Issuer: %s", connid, idx, cert.Issuer.String())
		d.Logger.Debugf("(conn #%d) %d: AltDNSNames: %+v", connid, idx, cert.DNSNames)
		d.Logger.Debugf("(conn #%d) %d: AltIPAddresses: %+v", connid, idx, cert.IPAddresses)
	}
	conn = &tlsConnWrapper{Conn: tlsconn, sessID: connid}
	return
}
