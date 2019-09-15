// Package netx contains OONI's net extensions.
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
	"github.com/bassosimone/netx/log"
)

// OperationID is the ID of a network-level operation.
type OperationID string

const (
	// CloseOperation is a close operation.
	CloseOperation = OperationID("close")

	// ConnectOperation is a connect operation.
	ConnectOperation = OperationID("connect")

	// ReadFromOperation is a readFrom operation.
	ReadFromOperation = OperationID("readFrom")

	// ReadOperation is a read operation.
	ReadOperation = OperationID("read")

	// ResolveOperation is a DNS-resolve operation.
	ResolveOperation = OperationID("resolve")

	// TLSHandsakeOperation is the TLS-handshake operation.
	TLSHandshakeOperation = OperationID("tlsHandshake")

	// WriteOperation is a write operation.
	WriteOperation = OperationID("write")

	// WriteToOperation is a writeTo operation.
	WriteToOperation = OperationID("writeTo")
)

// TimingMeasurement describes a timing measurement.
type TimingMeasurement struct {
	// Address is the address used by a ConnectOperation.
	Address string `json:",omitempty"`

	// Addresses contains the addresses returned by a ResolveOperation.
	Addresses []string `json:",omitempty"`

	// Data is the data sent or received by this operation.
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

	// NumBytes is the number of bytes transferred by the operation.
	NumBytes int64 `json:",omitempty"`

	// OperationID is the operation's ID..
	OperationID OperationID

	// SNI is the TLS server name indication being used.
	SNI string `json:",omitempty"`

	// SrcAddress string is WriteToOperation's source address.
	SrcAddress string `json:",omitempty"`

	// SessionID is the ID of a network level session initiated by
	// dialing and concluded by a dial failure or the closing of an
	// open net.Conn instance returned by dial.
	SessionID int64

	// StartTime is the time when the operaton started relative to the
	// moment stored in MeasuringDialer.Beginning.
	StartTime time.Duration

	// TLSConnectionState contains the TLS connection state.
	TLSConnectionState *tls.ConnectionState `json:",omitempty"`
}

// MeasuringDialer creates connections and keeps track of stats.
type MeasuringDialer struct {
	// net.Dialer is the base struct.
	net.Dialer

	// Beginning is the point in time considered as the beginning
	// of the measurements performed by this dialer.
	Beginning time.Time

	// BytesRead counts the bytes read by all the connections created
	// using this specific dialer.
	BytesRead int64

	// BytesWritten counts the bytes written by all the connections
	// created using this specific dialer.
	BytesWritten int64

	// EnableTiming controls whether to enable timing measurements. If
	// timing is enabled, then TimingMeasurements will be filled.
	EnableTiming bool

	// Logger is the interface used for logging.
	Logger log.Logger

	// LookupHost is the function called to perform host lookups by this
	// dialer. By default uses the embedded Dialer's resolver. To implement
	// e.g. DoT or DoH, override this function.
	LookupHost func(ctx context.Context, host string) (addrs []string, err error)

	// TimingMeasurements contains timing measurements. They are only saved
	// when the EnableTiming setting is true.
	TimingMeasurements []TimingMeasurement

	sessID int64
	mutex  sync.Mutex
}

// NewMeasuringDialer returns a new MeasuringDialer instance. The |beginning|
// argument is the time considered as the beginning of the measurements.
func NewMeasuringDialer(beginning time.Time) (d *MeasuringDialer) {
	d = new(MeasuringDialer)
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

func (d *MeasuringDialer) append(m TimingMeasurement) {
	d.mutex.Lock()
	d.TimingMeasurements = append(d.TimingMeasurements, m)
	d.mutex.Unlock()
}

type measurableConn struct {
	net.Conn
	dialer      *MeasuringDialer
	includeData bool
	sessID      int64
}

func (c *measurableConn) Read(b []byte) (n int, err error) {
	var start time.Time
	if c.dialer.EnableTiming {
		start = time.Now()
	}
	n, err = c.Conn.Read(b)
	if n > 0 {
		atomic.AddInt64(&c.dialer.BytesRead, int64(n))
	}
	if c.dialer.EnableTiming {
		c.dialer.Logger.Debugf("(conn #%d) read %d bytes", c.sessID, n)
		m := TimingMeasurement{
			Duration:    time.Now().Sub(start),
			Error:       err,
			NumBytes:    int64(n),
			OperationID: ReadOperation,
			SessionID:   c.sessID,
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
	if c.dialer.EnableTiming {
		start = time.Now()
	}
	n, err = c.Conn.Write(b)
	if n > 0 {
		atomic.AddInt64(&c.dialer.BytesWritten, int64(n))
	}
	if c.dialer.EnableTiming {
		c.dialer.Logger.Debugf("(conn #%d) written %d bytes", c.sessID, n)
		m := TimingMeasurement{
			Duration:    time.Now().Sub(start),
			Error:       err,
			NumBytes:    int64(n),
			OperationID: WriteOperation,
			SessionID:   c.sessID,
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
	var start time.Time
	if c.dialer.EnableTiming {
		start = time.Now()
	}
	err = c.Conn.Close()
	if c.dialer.EnableTiming {
		c.dialer.Logger.Debugf("(conn #%d) close", c.sessID)
		c.dialer.append(TimingMeasurement{
			Duration:    time.Now().Sub(start),
			Error:       err,
			OperationID: CloseOperation,
			SessionID:   c.sessID,
			StartTime:   start.Sub(c.dialer.Beginning),
		})
	}
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
	if c.dialer.EnableTiming {
		start = time.Now()
	}
	n, addr, err = packetConn.ReadFrom(p)
	if n > 0 {
		atomic.AddInt64(&c.dialer.BytesRead, int64(n))
	}
	if c.dialer.EnableTiming {
		m := TimingMeasurement{
			DestAddress: c.LocalAddr().String(),
			Duration:    time.Now().Sub(start),
			Error:       err,
			NumBytes:    int64(n),
			OperationID: ReadFromOperation,
			SessionID:   c.sessID,
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
	if c.dialer.EnableTiming {
		start = time.Now()
	}
	n, err = packetConn.WriteTo(p, addr)
	if n > 0 {
		atomic.AddInt64(&c.dialer.BytesRead, int64(n))
	}
	if c.dialer.EnableTiming {
		m := TimingMeasurement{
			DestAddress: addr.String(),
			Duration:    time.Now().Sub(start),
			Error:       err,
			NumBytes:    int64(n),
			OperationID: WriteToOperation,
			SessionID:   c.sessID,
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

// GetConnID returns the connection's ID.
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

// ErrDialContextTimeout is an error indicating that the context timed out
// while we were waiting for our dial attempts to complete.
type ErrDialContextTimeout struct {
	// Errors contains the list of errors that occurred.
	Errors []error
}

func (ErrDialContextTimeout) Error() string {
	return "netx.DialContext: context deadline expired"
}

// Timeout tells you whether this error is a timeout.
func (ErrDialContextTimeout) Timeout() bool {
	return true
}

// Temporary tells you whether this error is temporary.
func (ErrDialContextTimeout) Temporary() bool {
	return false
}

// TODO(bassosimone): we need to calibrate these parameters.
const (
	initialMean = 0.5
	finalMean   = 8.0
	meanFactor  = 2.0
	stdevFactor = 0.05
)

// Dial calls d.DialContext with the background context.
func (d *MeasuringDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialContext extends net.Dialer.DialContext to implement exponential backoff
// between retries, and optionally measure network events.
func (d *MeasuringDialer) DialContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	return d.dialContextEx(ctx, network, address, false)
}

func (d *MeasuringDialer) dialContextEx(
	ctx context.Context, network, address string, includeData bool,
) (net.Conn, error) {
	var multierr ErrDialContextTimeout
	sessID := atomic.AddInt64(&d.sessID, 1)
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

// ErrManyConnectFailed is returned when repeated attempts at connecting
// cycling over all available addresses faled.
type ErrManyConnectFailed struct {
	Errors []error
}

func (ErrManyConnectFailed) Error() string {
	return "netx.DialContext: cannot connect any of the specified addresses"
}

// Timeout indicates whether this error is a timeout.
func (ErrManyConnectFailed) Timeout() bool {
	return false
}

// Temporary indicates whether this error is temporary.
func (ErrManyConnectFailed) Temporary() bool {
	return false
}

func (d *MeasuringDialer) dialContextDNS(
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
	var multierr ErrManyConnectFailed
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

func (d *MeasuringDialer) lookupHost(
	ctx context.Context, host string, id int64,
) (addrs []string, err error) {
	var start time.Time
	if d.EnableTiming {
		start = time.Now()
	}
	addrs, err = d.LookupHost(ctx, host)
	if d.EnableTiming {
		d.append(TimingMeasurement{
			Addresses:   addrs,
			Duration:    time.Now().Sub(start),
			Error:       err,
			Hostname:    host,
			OperationID: ResolveOperation,
			SessionID:   id,
			StartTime:   start.Sub(d.Beginning),
		})
	}
	if err != nil {
		d.Logger.Debug(err.Error())
	}
	return
}

func (d *MeasuringDialer) dialContextAddrPort(
	ctx context.Context, network, addr, port string, id int64, includeData bool,
) (net.Conn, error) {
	var start time.Time
	if d.EnableTiming {
		start = time.Now()
	}
	// Assumption: dial using an IP address boils down to connect
	if net.ParseIP(addr) == nil {
		return nil, errors.New("dialContextAddrPort: expected an address")
	}
	endpoint := net.JoinHostPort(addr, port)
	d.Logger.Debugf("(conn #%d) connect %s/%s", id, endpoint, network)
	conn, err := d.Dialer.DialContext(ctx, network, endpoint)
	if d.EnableTiming {
		d.append(TimingMeasurement{
			Address:     endpoint,
			Duration:    time.Now().Sub(start),
			Error:       err,
			Network:     network,
			OperationID: ConnectOperation,
			SessionID:   id,
			StartTime:   start.Sub(d.Beginning),
		})
	}
	if err != nil {
		d.Logger.Debug(err.Error())
		return nil, err
	}
	return d.wrapConn(conn, id, includeData), nil
}

func (d *MeasuringDialer) wrapConn(conn net.Conn, id int64, includeData bool) net.Conn {
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

// DialTLS dials a tls.Conn connection using the specified tls.Config,
// handshake timeout, network, and address.
func (d *MeasuringDialer) DialTLS(
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
	var start time.Time
	if d.EnableTiming {
		start = time.Now()
	}
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
	if d.EnableTiming {
		d.append(TimingMeasurement{
			Duration:           time.Now().Sub(start),
			Error:              err,
			OperationID:        TLSHandshakeOperation,
			SNI:                config.ServerName,
			SessionID:          connid,
			StartTime:          start.Sub(d.Beginning),
			TLSConnectionState: &connstate,
		})
	}
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
