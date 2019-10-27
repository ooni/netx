// Package connx contains net.Conn extensions
package connx

import (
	"net"
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
			ConnID:                 c.ID,
			DurationSinceBeginning: stop.Sub(c.Beginning),
			Error:                  err,
			NumBytes:               int64(n),
			SyscallDuration:        stop.Sub(start),
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
			ConnID:                 c.ID,
			DurationSinceBeginning: stop.Sub(c.Beginning),
			Error:                  err,
			NumBytes:               int64(n),
			SyscallDuration:        stop.Sub(start),
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
			ConnID:                 c.ID,
			DurationSinceBeginning: stop.Sub(c.Beginning),
			Error:                  err,
			SyscallDuration:        stop.Sub(start),
		},
	})
	return
}
