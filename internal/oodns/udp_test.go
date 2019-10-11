package oodns

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestIntegrationUDPSuccess(t *testing.T) {
	transport := NewTransportUDP(
		"9.9.9.9:53", (&net.Dialer{}).DialContext,
	)
	err := threeRounds(transport)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationDialContextExFailure(t *testing.T) {
	transport := &udpTransport{
		address:     "9.9.9.9:53",
		dialContext: (&net.Dialer{}).DialContext,
	}
	transport.dialContext = func(
		ctx context.Context, network string, address string,
	) (net.Conn, error) {
		return nil, errors.New("mocked error")
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationSetDeadlineError(t *testing.T) {
	transport := &udpTransport{
		address:     "9.9.9.9:53",
		dialContext: (&net.Dialer{}).DialContext,
	}
	transport.dialContext = func(
		ctx context.Context, network string, address string,
	) (net.Conn, error) {
		return fakeconn{
			setDeadlineError: errors.New("mocked error"),
		}, nil
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationWriteError(t *testing.T) {
	transport := &udpTransport{
		address:     "9.9.9.9:53",
		dialContext: (&net.Dialer{}).DialContext,
	}
	transport.dialContext = func(
		ctx context.Context, network string, address string,
	) (net.Conn, error) {
		return fakeconn{
			writeError: errors.New("mocked error"),
		}, nil
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

type fakeconn struct {
	setDeadlineError error
	writeError       error
}

func (fakeconn) Read(b []byte) (n int, err error) {
	n = len(b)
	return
}
func (c fakeconn) Write(b []byte) (n int, err error) {
	n, err = len(b), c.writeError
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
func (c fakeconn) SetDeadline(t time.Time) error {
	return c.setDeadlineError
}
func (c fakeconn) SetReadDeadline(t time.Time) error {
	return c.SetDeadline(t)
}
func (c fakeconn) SetWriteDeadline(t time.Time) error {
	return c.SetDeadline(t)
}
