package dnsovertcp_test

import (
	"crypto/tls"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/dnstransport/dnsovertcp"
)

func TestIntegrationSuccess(t *testing.T) {
	transport := dnsovertcp.NewTransport(
		time.Now(), handlers.NoHandler, "dns.quad9.net",
	)
	if err := threeRounds(transport); err != nil {
		t.Fatal(err)
	}
	transport = dnsovertcp.NewTransport(
		time.Now(), handlers.NoHandler, "9.9.9.9",
	)
	transport.NoTLS = true
	if err := threeRounds(transport); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationLookupHostError(t *testing.T) {
	transport := dnsovertcp.NewTransport(
		time.Now(), handlers.NoHandler, "antani.local",
	)
	if err := roundTrip(transport, "ooni.io."); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationCustomTLSConfig(t *testing.T) {
	transport := dnsovertcp.NewTransport(
		time.Now(), handlers.NoHandler, "dns.quad9.net",
	)
	transport.Dialer.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if err := roundTrip(transport, "ooni.io."); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationDialFailure(t *testing.T) {
	transport := dnsovertcp.NewTransport(
		time.Now(), handlers.NoHandler, "dns.quad9.net",
	)
	transport.Port = "53" // should cause dial to fail
	if err := roundTrip(transport, "ooni.io."); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationLookupHostFailure(t *testing.T) {
	transport := dnsovertcp.NewTransport(
		time.Now(), handlers.NoHandler, "dns.quad9.net",
	)
	transport.LookupHost = func(host string) ([]string, error) {
		return nil, errors.New("mocked error")
	}
	if err := roundTrip(transport, "ooni.io."); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationEmptyLookupReply(t *testing.T) {
	transport := dnsovertcp.NewTransport(
		time.Now(), handlers.NoHandler, "dns.quad9.net",
	)
	transport.LookupHost = func(host string) ([]string, error) {
		return nil, nil
	}
	if err := roundTrip(transport, "ooni.io."); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestUnitRoundTripWithConnFailure(t *testing.T) {
	transport := dnsovertcp.NewTransport(
		time.Now(), handlers.NoHandler, "dns.quad9.net",
	)
	query := make([]byte, 1<<10)
	// fakeconn will fail in the SetDeadline, therefore we will have
	// an immediate error and we expect all errors the be alike
	reply, err := transport.RoundTripWithConn(&fakeconn{}, query)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if reply != nil {
		t.Fatal("expected nil error here")
	}
}

func threeRounds(transport *dnsovertcp.Transport) error {
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

func roundTrip(transport *dnsovertcp.Transport, domain string) error {
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
