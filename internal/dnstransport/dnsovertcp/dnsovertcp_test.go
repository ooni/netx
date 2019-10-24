package dnsovertcp

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func dialTLS(config *tls.Config) func(network, address string) (net.Conn, error) {
	return func(network, address string) (net.Conn, error) {
		return tls.Dial(network, address, config)
	}
}

func dialTCP(network, address string) (net.Conn, error) {
	return net.Dial(network, address)
}

func TestIntegrationSuccessTLS(t *testing.T) {
	// "Dial interprets a nil configuration as equivalent to
	// the zero configuration; see the documentation of Config
	// for the defaults."
	transport := NewTransport(dialTLS(nil), "dns.quad9.net:853")
	if err := threeRounds(transport); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationSuccessTCP(t *testing.T) {
	transport := NewTransport(dialTCP, "9.9.9.9:53")
	if err := threeRounds(transport); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationLookupHostError(t *testing.T) {
	transport := NewTransport(dialTCP, "antani.local")
	if err := roundTrip(transport, "ooni.io."); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationCustomTLSConfig(t *testing.T) {
	transport := NewTransport(dialTLS(&tls.Config{
		MinVersion: tls.VersionTLS10,
	}), "dns.quad9.net:853")
	if err := roundTrip(transport, "ooni.io."); err != nil {
		t.Fatal(err)
	}
}

func TestUnitRoundTripWithConnFailure(t *testing.T) {
	// fakeconn will fail in the SetDeadline, therefore we will have
	// an immediate error and we expect all errors the be alike
	transport := NewTransport(func(network, address string) (net.Conn, error) {
		return &fakeconn{}, nil
	}, "8.8.8.8:53")
	query := make([]byte, 1<<10)
	reply, err := transport.doWithConn(&fakeconn{}, query)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if reply != nil {
		t.Fatal("expected nil error here")
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
	data, err = transport.RoundTrip(context.Background(), data)
	if err != nil {
		return err
	}
	return query.Unpack(data)
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
	return errors.New("cannot set deadline")
}
func (fakeconn) SetReadDeadline(t time.Time) (err error) {
	return
}
func (fakeconn) SetWriteDeadline(t time.Time) (err error) {
	return
}
