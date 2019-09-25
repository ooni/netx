// Package connx contains net.Conn extensions
package connx

import (
	"net"
	"syscall"
	"time"

	"github.com/ooni/netx/model"
)

// MeasuringConn is a net.Conn used to perform measurements
type MeasuringConn struct {
	net.Conn
	Beginning time.Time
	Handler   model.Handler
	ID        int64
}

// Read reads data from the connection.
func (c *MeasuringConn) Read(b []byte) (n int, err error) {
	start := time.Now()
	n, err = c.Conn.Read(b)
	stop := time.Now()
	c.Handler.OnMeasurement(model.Measurement{
		Read: &model.ReadEvent{
			Duration: stop.Sub(start),
			Error:    err,
			NumBytes: int64(n),
			ConnID:   c.ID,
			Time:     stop.Sub(c.Beginning),
		},
	})
	return
}

// Write writes data to the connection
func (c *MeasuringConn) Write(b []byte) (n int, err error) {
	start := time.Now()
	n, err = c.Conn.Write(b)
	stop := time.Now()
	c.Handler.OnMeasurement(model.Measurement{
		Write: &model.WriteEvent{
			Duration: stop.Sub(start),
			Error:    err,
			NumBytes: int64(n),
			ConnID:   c.ID,
			Time:     stop.Sub(c.Beginning),
		},
	})
	return
}

// Close closes the connection
func (c *MeasuringConn) Close() (err error) {
	start := time.Now()
	err = c.Conn.Close()
	stop := time.Now()
	c.Handler.OnMeasurement(model.Measurement{
		Close: &model.CloseEvent{
			Duration: stop.Sub(start),
			Error:    err,
			ConnID:   c.ID,
			Time:     stop.Sub(c.Beginning),
		},
	})
	return
}

// DNSMeasuringConn is like MeasuringConn except that it also
// implements the net.PacketConn interface. This is required
// to convince the Go resolver that this is an UDP connection.
type DNSMeasuringConn struct {
	MeasuringConn
}

// Read reads data from the connection.
func (c *DNSMeasuringConn) Read(b []byte) (n int, err error) {
	n, err = c.MeasuringConn.Read(b)
	if err == nil {
		c.MeasuringConn.Handler.OnMeasurement(model.Measurement{
			DNSReply: &model.DNSReplyEvent{
				ConnID: c.MeasuringConn.ID,
				Message: model.DNSMessage{
					Data: b[:n],
				},
				Time: time.Now().Sub(c.MeasuringConn.Beginning),
			},
		})
	}
	return
}

// ReadFrom reads from the PacketConn.
func (c *DNSMeasuringConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	// This is just a not implemented stub.
	err = net.Error(&net.OpError{
		Op:     "ReadFrom",
		Source: c.Conn.LocalAddr(),
		Addr:   c.Conn.RemoteAddr(),
		Err:    syscall.ENOTCONN,
	})
	return
}

// Write writes data to the connection
func (c *DNSMeasuringConn) Write(b []byte) (n int, err error) {
	n, err = c.MeasuringConn.Write(b)
	if err == nil {
		c.MeasuringConn.Handler.OnMeasurement(model.Measurement{
			DNSQuery: &model.DNSQueryEvent{
				ConnID: c.MeasuringConn.ID,
				Message: model.DNSMessage{
					Data: b,
				},
				Time: time.Now().Sub(c.MeasuringConn.Beginning),
			},
		})
	}
	return
}

// WriteTo writes to the PacketConn.
func (c *DNSMeasuringConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	// This is just a not implemented stub.
	err = net.Error(&net.OpError{
		Op:     "WriteTo",
		Source: c.Conn.LocalAddr(),
		Addr:   c.Conn.RemoteAddr(),
		Err:    syscall.ENOTCONN,
	})
	return
}
