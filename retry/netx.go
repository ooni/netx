// Package netx contains OONI's net extensions.
//
// This package provides a replacement for net.Dialer that can Dial,
// DialContext, and DialTLS. During its lifecycle this modified Dialer
// will observe network level events and collect TimingMeasurements.
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
// TimeMeasurement is the data structure used by all the network
// level events we collect. The OperationID of a specific
// TimeMeasurement structure determines what fields of such
// structure are meaningful.
package netx

import (
	"context"
	"crypto/tls"
	"errors"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
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

	// ReadFromOperation is the ID of net.PacketConn.ReadFrom.
	ReadFromOperation = OperationID("readFrom")

	// ReadOperation is the ID of net.Conn.Read.
	ReadOperation = OperationID("read")

	// ResolveOperation is the ID of net.Resolver.LookupHost.
	ResolveOperation = OperationID("resolve")

	// TLSHandshakeOperation is the ID of tls.Conn.Handshake.
	TLSHandshakeOperation = OperationID("tlsHandshake")

	// WriteOperation is the ID of net.Conn.Write.
	WriteOperation = OperationID("write")

	// WriteToOperation is the ID of net.PacketConn.WriteTo.
	WriteToOperation = OperationID("writeTo")
)

// TimingMeasurement describes a timing measurement. Some fields
// should be always present, others are optional. The optional
// fields are also marked with `json:",omitempty"`.
type TimingMeasurement struct {
	// Address is the address used by a ConnectOperation.
	Address string `json:",omitempty"`

	// Addresses contains the addresses returned by a ResolveOperation.
	Addresses []string `json:",omitempty"`

	// ConnID is the ID of the connection. Note that the ID is assigned
	// when we start dialing, therefore, all connection attempts part
	// of a single dial operation will have the same ID.
	ConnID int64

	// Data is the data transferred by ReadFromOperation, ReadOperation,
	// WriteOperation, WriteToOperation. Note that this field will be
	// empty unless Dialer.EnableFullTiming is true.
	Data []byte `json:",omitempty"`

	// DestAddress is WriteToOperation's destination address.
	DestAddress string `json:",omitempty"`

	// Duration is the operation's duration.
	Duration time.Duration

	// Error is the error that occurred, or nil.
	Error error

	// Hostname is the hostname passed to a ResolveOperation.
	Hostname string `json:",omitempty"`

	// Network is the network used by a ConnectOperation.
	Network string `json:",omitempty"`

	// NumBytes is the number of bytes transferred by ReadFromOperation,
	// ReadOperation, WriteOperation, WriteToOperation.
	NumBytes int64 `json:",omitempty"`

	// OperationID is the operation's ID.
	OperationID OperationID

	// SNI is the SNI used by TLSHandshakeOperation.
	SNI string `json:",omitempty"`

	// SrcAddress string is WriteToOperation's source address.
	SrcAddress string `json:",omitempty"`

	// StartTime is the time when the operaton started relative to the
	// moment stored in Dialer.Beginning.
	StartTime time.Duration

	// TLSConnectionState is the TLS state of TLSHandshakeOperation.
	TLSConnectionState *tls.ConnectionState `json:",omitempty"`
}

// Dialer creates connections and collects TimingMeasurements.
type Dialer struct {
	// net.Dialer is the base struct.
	net.Dialer

	// Beginning is the point in time considered as the beginning
	// of the measurements performed by this dialer. This field is
	// initialized by the NewDialer constructor.
	Beginning time.Time

	// BytesRead counts the bytes read by all the connections created
	// using this specific dialer.
	BytesRead int64

	// BytesWritten counts the bytes written by all the connections
	// created using this specific dialer.
	BytesWritten int64

	// EnableFullTiming controls whether to also measure the timing of
	// I/O operations. By default, we only measure the timing of the
	// connect and TLS handshake operations. By setting this field to
	// true, you enable also timing Read and Write operations.
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

	// TimingMeasurements contains timing measurements. They are only saved
	// when the EnableTiming setting is true.
	TimingMeasurements []TimingMeasurement

	connID int64
	mutex  sync.Mutex
}

// NewDialer returns a new Dialer instance.
func NewDialer(beginning time.Time) (d *Dialer) {
	d = new(Dialer)
	d.Beginning = beginning
	d.LookupHost = func(ctx context.Context, host string) (addrs []string, err error) {
		return d.Dialer.Resolver.LookupHost(ctx, host)
	}
	d.Dialer = net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return d.dialContextEx(ctx, network, address, true)
			},
		},
	}
	d.Logger = &internal.NoLogger{}
	return
}

// PopMeasurements extracts the measurements collected by this dialer and
// returns them in a goroutine safe way.
func (d *Dialer) PopMeasurements() (measurements []TimingMeasurement) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	measurements = d.TimingMeasurements
	d.TimingMeasurements = nil
	return
}

func (d *Dialer) append(m TimingMeasurement) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.TimingMeasurements = append(d.TimingMeasurements, m)
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
	if c.dialer.EnableFullTiming {
		c.dialer.Logger.Debugf("(conn #%d) read %d bytes", c.sessID, n)
		m := TimingMeasurement{
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
	}
	if err != nil {
		c.dialer.Logger.Debugf("(conn #%d) %s", c.sessID, err.Error())
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
	if c.dialer.EnableFullTiming {
		c.dialer.Logger.Debugf("(conn #%d) written %d bytes", c.sessID, n)
		m := TimingMeasurement{
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
	}
	if err != nil {
		c.dialer.Logger.Debugf("(conn #%d) %s", c.sessID, err.Error())
	}
	return
}

func (c *measurableConn) Close() (err error) {
	start := time.Now()
	err = c.Conn.Close()
	c.dialer.Logger.Debugf("(conn #%d) close", c.sessID)
	c.dialer.append(TimingMeasurement{
		Duration:    time.Now().Sub(start),
		Error:       err,
		OperationID: CloseOperation,
		ConnID:      c.sessID,
		StartTime:   start.Sub(c.dialer.Beginning),
	})
	if err != nil {
		c.dialer.Logger.Debugf("(conn #%d) %s", c.sessID, err.Error())
	}
	return
}

// measurablePacketConn is required by Go's dnsclient, which behaves
// differently depending on the type of connection.
type measurablePacketConn struct {
	measurableConn
}

func (c *measurablePacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	packetConn := c.Conn.(net.PacketConn)
	var start time.Time
	if c.dialer.EnableFullTiming {
		start = time.Now()
	}
	n, addr, err = packetConn.ReadFrom(p)
	if n > 0 {
		atomic.AddInt64(&c.dialer.BytesRead, int64(n))
	}
	if c.dialer.EnableFullTiming {
		m := TimingMeasurement{
			DestAddress: c.LocalAddr().String(),
			Duration:    time.Now().Sub(start),
			Error:       err,
			NumBytes:    int64(n),
			OperationID: ReadFromOperation,
			ConnID:      c.sessID,
			SrcAddress:  addr.String(),
			StartTime:   start.Sub(c.dialer.Beginning),
		}
		if c.includeData && n > 0 {
			m.Data = p[:n]
		}
		c.dialer.append(m)
	}
	return
}

func (c *measurablePacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	packetConn := c.Conn.(net.PacketConn)
	var start time.Time
	if c.dialer.EnableFullTiming {
		start = time.Now()
	}
	n, err = packetConn.WriteTo(p, addr)
	if n > 0 {
		atomic.AddInt64(&c.dialer.BytesRead, int64(n))
	}
	if c.dialer.EnableFullTiming {
		m := TimingMeasurement{
			DestAddress: addr.String(),
			Duration:    time.Now().Sub(start),
			Error:       err,
			NumBytes:    int64(n),
			OperationID: WriteToOperation,
			ConnID:      c.sessID,
			SrcAddress:  c.Conn.LocalAddr().String(),
			StartTime:   start.Sub(c.dialer.Beginning),
		}
		if c.includeData && n > 0 {
			m.Data = p[:n]
		}
		c.dialer.append(m)
	}
	return
}

// GetConnID returns the connection's unique identifier. If this is not a
// connection we created, this function returns false. Otherwise, it returns
// true and `*id` will contain the unique connection identifier.
func GetConnID(conn net.Conn, id *int64) bool {
	if c, ok := conn.(*measurableConn); ok {
		*id = c.sessID
		return true
	}
	if c, ok := conn.(*measurablePacketConn); ok {
		*id = c.sessID
		return true
	}
	if c, ok := conn.(*tlsConnWrapper); ok {
		*id = c.sessID
		return true
	}
	return false
}

type errDialContextTimeout struct {
	Errors []error
}

func (errDialContextTimeout) Error() string {
	return "netx.DialContext: context deadline expired"
}
func (errDialContextTimeout) Timeout() bool {
	return true
}
func (errDialContextTimeout) Temporary() bool {
	return false
}

// TODO(bassosimone): we need to calibrate these parameters.
const (
	initialMean = 0.5
	finalMean   = 8.0
	meanFactor  = 2.0
	stdevFactor = 0.05
)

// Dial creates a TCP or UDP connection.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialContext is like Dial but the context allows to interrupt a
// pending connection attempt early.
func (d *Dialer) DialContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	return d.dialContextEx(ctx, network, address, false)
}

func (d *Dialer) dialContextEx(
	ctx context.Context, network, address string, includeData bool,
) (net.Conn, error) {
	var multierr errDialContextTimeout
	sessID := atomic.AddInt64(&d.connID, 1)
	onfailure := func() (net.Conn, error) {
		err := net.Error(multierr)
		d.Logger.Debugf("(conn #%d) %s", sessID, err.Error())
		return nil, err
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for mean := initialMean; mean <= finalMean; mean *= meanFactor {
		conn, err := d.dialContextDNS(ctx, network, address, sessID, includeData)
		if err == nil {
			return conn, nil
		}
		multierr.Errors = append(multierr.Errors, err)
		// Now backoff
		stdev := stdevFactor * mean
		seconds := rng.NormFloat64()*stdev + mean
		sleepTime := time.Duration(seconds * float64(time.Second))
		d.Logger.Debugf("(conn #%d) retrying in %s", sessID, sleepTime.String())
		timer := time.NewTimer(sleepTime)
		select {
		case <-ctx.Done():
			timer.Stop()
			multierr.Errors = append(multierr.Errors, ctx.Err())
			return onfailure()
		case <-timer.C:
			// FALLTHROUGH
		}
	}
	return onfailure()
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

func (d *Dialer) dialContextDNS(
	ctx context.Context, network, address string, id int64, includeData bool,
) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if net.ParseIP(host) != nil {
		return d.dialContextAddrPort(ctx, network, host, port, id, includeData)
	}
	addrs, err := d.lookupHost(ctx, host, id)
	if err != nil {
		return nil, err
	}
	var multierr errManyConnectFailed
	for _, addr := range addrs {
		conn, err := d.dialContextAddrPort(ctx, network, addr, port, id, includeData)
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

func (d *Dialer) lookupHost(
	ctx context.Context, host string, id int64,
) (addrs []string, err error) {
	start := time.Now()
	addrs, err = d.LookupHost(ctx, host)
	d.append(TimingMeasurement{
		Addresses:   addrs,
		Duration:    time.Now().Sub(start),
		Error:       err,
		Hostname:    host,
		OperationID: ResolveOperation,
		ConnID:      id,
		StartTime:   start.Sub(d.Beginning),
	})
	if err != nil {
		d.Logger.Debug(err.Error())
	}
	return
}

func (d *Dialer) dialContextAddrPort(
	ctx context.Context, network, addr, port string, id int64, includeData bool,
) (net.Conn, error) {
	start := time.Now()
	// Assumption: dial using an IP address boils down to connect
	if net.ParseIP(addr) == nil {
		return nil, errors.New("dialContextAddrPort: expected an address")
	}
	endpoint := net.JoinHostPort(addr, port)
	d.Logger.Debugf("(conn #%d) connect %s/%s", id, endpoint, network)
	conn, err := d.Dialer.DialContext(ctx, network, endpoint)
	d.append(TimingMeasurement{
		Address:     endpoint,
		Duration:    time.Now().Sub(start),
		Error:       err,
		Network:     network,
		OperationID: ConnectOperation,
		ConnID:      id,
		StartTime:   start.Sub(d.Beginning),
	})
	if err != nil {
		d.Logger.Debug(err.Error())
		return nil, err
	}
	return d.wrapConn(conn, id, includeData), nil
}

func (d *Dialer) wrapConn(conn net.Conn, id int64, includeData bool) net.Conn {
	if _, ok := conn.(net.PacketConn); ok {
		// When a connection is a PacketConn, make sure we return a
		// structure that matches the PacketConn interface.
		return &measurablePacketConn{
			measurableConn: measurableConn{
				Conn:        conn,
				dialer:      d,
				includeData: includeData,
				sessID:      id,
			},
		}
	}
	return &measurableConn{
		Conn:        conn,
		dialer:      d,
		includeData: includeData,
		sessID:      id,
	}
}

type tlsConnWrapper struct {
	net.Conn
	sessID int64
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, addr string) (conn net.Conn, err error) {
	return
}

// DialTLSEx is a deprecated function you should not use.
func (d *Dialer) DialTLSEx(
	config *tls.Config, handshakeTimeout time.Duration, network, addr string,
) (conn net.Conn, err error) {
	conn, err = d.Dial(network, addr)
	if err != nil {
		return
	}
	var connid int64
	if GetConnID(conn, &connid) == false {
		conn.Close()
		return nil, errors.New("netx: unexpectedly missing connid")
	}
	if config == nil {
		hostname, _, err := net.SplitHostPort(addr)
		if err != nil {
			conn.Close()
			return nil, err
		}
		config = &tls.Config{ServerName: hostname}
	}
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
	d.append(TimingMeasurement{
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
		conn.Close()
		return nil, err
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
