// Package connx contains net.Conn extensions
package connx

import (
	"net"
	"time"

	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
)

// MeasuringConn is a net.Conn used to perform measurements
type MeasuringConn struct {
	net.Conn
	info *tracing.Info
}

// NewMeasuringConn creates a new measuring conn.
func NewMeasuringConn(conn net.Conn, info *tracing.Info) *MeasuringConn {
	return &MeasuringConn{
		Conn: conn,
		info: info,
	}
}

// Read reads data from the connection.
func (c *MeasuringConn) Read(b []byte) (n int, err error) {
	start := time.Now()
	n, err = c.Conn.Read(b)
	stop := time.Now()
	c.info.Handler.OnMeasurement(model.Measurement{
		Read: &model.ReadEvent{
			SyscallEvent: model.SyscallEvent{
				BaseEvent:   c.info.BaseEvent(),
				BlockedTime: stop.Sub(start),
			},
			Error:    err,
			NumBytes: int64(n),
		},
	})
	return
}

// Write writes data to the connection
func (c *MeasuringConn) Write(b []byte) (n int, err error) {
	start := time.Now()
	n, err = c.Conn.Write(b)
	stop := time.Now()
	c.info.Handler.OnMeasurement(model.Measurement{
		Write: &model.WriteEvent{
			SyscallEvent: model.SyscallEvent{
				BaseEvent:   c.info.BaseEvent(),
				BlockedTime: stop.Sub(start),
			},
			Error:    err,
			NumBytes: int64(n),
		},
	})
	return
}
