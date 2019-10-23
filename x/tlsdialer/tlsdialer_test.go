package tlsdialer

import (
	"crypto/tls"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
)

func TestIntegrationSuccess(t *testing.T) {
	dialer := newdialer()
	conn, err := dialer.DialTLS("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("connection is nil")
	}
	conn.Close()
}

func TestIntegrationFailureSplitHostPort(t *testing.T) {
	dialer := newdialer()
	conn, err := dialer.DialTLS("tcp", "www.google.com") // missing port
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func TestIntegrationFailureConnectTimeout(t *testing.T) {
	dialer := newdialer()
	dialer.ConnectTimeout = 10 * time.Microsecond
	conn, err := dialer.DialTLS("tcp", "www.google.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func TestIntegrationFailureTLSHandshakeTimeout(t *testing.T) {
	dialer := newdialer()
	dialer.TLSHandshakeTimeout = 10 * time.Microsecond
	conn, err := dialer.DialTLS("tcp", "www.google.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func TestIntegrationFailureSetDeadline(t *testing.T) {
	dialer := newdialer()
	dialer.setDeadline = func(conn net.Conn, t time.Time) error {
		return errors.New("mocked error")
	}
	conn, err := dialer.DialTLS("tcp", "www.google.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func newdialer() *TLSDialer {
	return New(
		time.Now(),
		handlers.NoHandler,
		new(net.Dialer),
		new(tls.Config),
	)
}
