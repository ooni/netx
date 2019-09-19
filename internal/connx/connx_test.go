package connx_test

import (
	"net"
	"testing"
	"time"

	"github.com/bassosimone/netx/handlers"
	"github.com/bassosimone/netx/internal/connx"
)

func TestIntegrationMeasuringConn(t *testing.T) {
	conn := net.Conn(&connx.MeasuringConn{
		Conn:    fakeconn{},
		Handler: handlers.StdoutHandler,
	})
	defer conn.Close()
	data := make([]byte, 1<<17)
	n, err := conn.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(data) {
		t.Fatal("invalid number of bytes read")
	}
	n, err = conn.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(data) {
		t.Fatal("invalid number of bytes written")
	}
}

func TestIntegrationDNSMeasuringConn(t *testing.T) {
	conn := net.Conn(&connx.DNSMeasuringConn{
		MeasuringConn: connx.MeasuringConn{
			Conn:    fakeconn{},
			Handler: handlers.StdoutHandler,
		},
	})
	defer conn.Close()
	data := make([]byte, 1<<17)
	n, err := conn.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(data) {
		t.Fatal("invalid number of bytes read")
	}
	n, err = conn.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(data) {
		t.Fatal("invalid number of bytes written")
	}
	packetconn := conn.(net.PacketConn)
	n, _, err = packetconn.ReadFrom(data)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if n != 0 {
		t.Fatal("expected zero here")
	}
	n, err = packetconn.WriteTo(data, &net.TCPAddr{})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if n != 0 {
		t.Fatal("expected zero here")
	}
}

type fakeconn struct{}

func (fakeconn) Read(b []byte) (n int, err error) {
	n = len(b)
	return
}
func (fakeconn) Write(b []byte) (n int, err error) {
	n = len(b)
	return
}
func (fakeconn) Close() (err error) {
	return
}
func (fakeconn) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}
func (fakeconn) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}
func (fakeconn) SetDeadline(t time.Time) (err error) {
	return
}
func (fakeconn) SetReadDeadline(t time.Time) (err error) {
	return
}
func (fakeconn) SetWriteDeadline(t time.Time) (err error) {
	return
}
