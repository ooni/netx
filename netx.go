// Package netx contains OONI's net extensions.
package netx

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// OperationID is the ID of a network-level operation.
type OperationID string

const (
	// CloseOperation is a close operation.
	CloseOperation = OperationID("close")

	// ConnectOperation is a connect operation.
	ConnectOperation = OperationID("connect")

	// ReadOperation is a read operation.
	ReadOperation = OperationID("read")

	// ResolveOperation is a DNS-resolve operation.
	ResolveOperation = OperationID("resolve")

	// WriteOperation is a write operation.
	WriteOperation = OperationID("write")
)

// TimingMeasurement describes a timing measurement.
type TimingMeasurement struct {
	// Address is the address used by a ConnectOperation.
	Address string `json:",omitempty"`

	// Addresses contains the addresses returned by a ResolveOperation.
	Addresses []string `json:",omitempty"`

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

	// OperationID is the operation OperationID.
	OperationID OperationID

	// SessionID is the ID of a network level session initiated by
	// trying to dial and concluded by either a dial error or by closing
	// a net.Conninstance returned by dial.
	SessionID int64

	// StartTime is the time when the operaton started relative to the
	// moment stored in MeasuringDialer.Beginning.
	StartTime time.Duration
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

	// LookupHost is the function called to perform host lookups by this
	// dialer. By default uses the embedded Dialer's resolver.
	LookupHost func(ctx context.Context, host string) (addrs []string, err error)

	// TimingMeasurements contains timing measurements.
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
	return
}

func (d *MeasuringDialer) append(m TimingMeasurement) {
	d.mutex.Lock()
	d.TimingMeasurements = append(d.TimingMeasurements, m)
	d.mutex.Unlock()
}

type measurableConn struct {
	net.Conn
	dialer *MeasuringDialer
	sessID int64
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
		c.dialer.append(TimingMeasurement{
			Duration:    time.Now().Sub(start),
			Error:       err,
			NumBytes:    int64(n),
			OperationID: ReadOperation,
			SessionID:   c.sessID,
			StartTime:   start.Sub(c.dialer.Beginning),
		})
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
		c.dialer.append(TimingMeasurement{
			Duration:    time.Now().Sub(start),
			Error:       err,
			NumBytes:    int64(n),
			OperationID: WriteOperation,
			SessionID:   c.sessID,
			StartTime:   start.Sub(c.dialer.Beginning),
		})
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
		c.dialer.append(TimingMeasurement{
			Duration:    time.Now().Sub(start),
			Error:       err,
			OperationID: CloseOperation,
			SessionID:   c.sessID,
			StartTime:   start.Sub(c.dialer.Beginning),
		})
	}
	return
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
func (d *MeasuringDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	var multierr ErrDialContextTimeout
	sessID := atomic.AddInt64(&d.sessID, 1)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for mean := initialMean; mean <= finalMean; mean *= meanFactor {
		conn, err := d.dialContextDNS(ctx, network, address, sessID)
		if err == nil {
			return conn, nil
		}
		multierr.Errors = append(multierr.Errors, err)
		// Now backoff
		stdev := stdevFactor * mean
		seconds := rng.NormFloat64()*stdev + mean
		timer := time.NewTimer(time.Duration(seconds * float64(time.Second)))
		select {
		case <-ctx.Done():
			timer.Stop()
			multierr.Errors = append(multierr.Errors, ctx.Err())
			return nil, net.Error(multierr)
		case <-timer.C:
			// FALLTHROUGH
		}
	}
	return nil, net.Error(multierr)
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
	ctx context.Context, network, address string, id int64,
) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if net.ParseIP(host) != nil {
		return d.dialContextAddrPort(ctx, network, host, port, id)
	}
	addrs, err := d.lookupHost(ctx, host, id)
	if err != nil {
		return nil, err
	}
	var multierr ErrManyConnectFailed
	for _, addr := range addrs {
		conn, err := d.dialContextAddrPort(ctx, network, addr, port, id)
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
	return
}

func (d *MeasuringDialer) dialContextAddrPort(
	ctx context.Context, network, addr, port string, id int64,
) (net.Conn, error) {
	var start time.Time
	if d.EnableTiming {
		start = time.Now()
	}
	// Assumption: a dial using an IP address boils down to connect
	if net.ParseIP(addr) == nil {
		return nil, errors.New("dialContextAddrPort: expected an address")
	}
	endpoint := net.JoinHostPort(addr, port)
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
		return nil, err
	}
	conn = &measurableConn{
		Conn:   conn,
		dialer: d,
		sessID: id,
	}
	return conn, err
}
