package dnsoverudp

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/connx"
)

var dialfunc = (&net.Dialer{}).DialContext

func TestIntegrationSuccessWithAddress(t *testing.T) {
	transport := NewTransport(
		dialfunc, "9.9.9.9:53",
	)
	err := threeRounds(transport)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationSuccessWithDomain(t *testing.T) {
	transport := NewTransport(
		dialfunc, "dns.quad9.net:53",
	)
	err := threeRounds(transport)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationDialFailure(t *testing.T) {
	transport := NewTransport(
		dialfunc, "9.9.9.9:53",
	)
	transport.dial = func(
		ctx context.Context, network, address string) (net.Conn, error) {
		return nil, errors.New("mocked error")
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationSetDeadlineError(t *testing.T) {
	transport := NewTransport(
		dialfunc, "9.9.9.9:53",
	)
	transport.dial = func(
		ctx context.Context, network, address string) (net.Conn, error) {
		return &connx.MeasuringConn{
			Conn: fakeconn{
				setDeadlineError: errors.New("mocked error"),
			},
			Handler: handlers.NoHandler,
		}, nil
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationWriteError(t *testing.T) {
	transport := NewTransport(
		dialfunc, "9.9.9.9:53",
	)
	transport.dial = func(
		ctx context.Context, network, address string) (net.Conn, error) {
		return &connx.MeasuringConn{
			Conn: fakeconn{
				writeError: errors.New("mocked error"),
			},
			Handler: handlers.NoHandler,
		}, nil
	}
	err := threeRounds(transport)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func threeRounds(transport *Transport) error {
	err := roundTrip(transport, "ooni.io.")
	if err != nil {
		return err
	}
	err = roundTrip(transport, "slashdot.org.")
	if err != nil {
		return err
	}
	err = roundTrip(transport, "kernel.org.")
	if err != nil {
		return err
	}
	return nil
}

func roundTrip(transport *Transport, domain string) error {
	query := new(dns.Msg)
	query.SetQuestion(domain, dns.TypeA)
	data, err := query.Pack()
	if err != nil {
		return err
	}
	data, err = transport.RoundTrip(data)
	if err != nil {
		return err
	}
	return query.Unpack(data)
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
