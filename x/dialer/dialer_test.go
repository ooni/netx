package dialer

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
)

func TestIntegrationSuccess(t *testing.T) {
	dialer := newdialer()
	conn, err := dialer.Dial("tcp", "www.google.com:80")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("nil connection")
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("GET / HTTP/1.0\r\n\r\n")); err != nil {
		t.Fatal(err)
	}
	buff := make([]byte, 1<<17)
	count, err := conn.Read(buff)
	if err != nil {
		t.Fatal(err)
	}
	if count <= 0 {
		t.Fatal("unexpected count")
	}
}

func TestIntegrationFailureSplitHostPort(t *testing.T) {
	dialer := newdialer()
	conn, err := dialer.Dial("tcp", "www.google.com") // missing port
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("non-nil connection")
	}
}

func TestIntegrationFailureLookupHost(t *testing.T) {
	dialer := newdialer()
	conn, err := dialer.Dial("tcp", "antani.local:443") // invalid host
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("non-nil connection")
	}
}

func TestIntegrationFailureDialAddress(t *testing.T) {
	dialer := newdialer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	conn, err := dialer.DialContext(
		ctx, "tcp", "8.8.8.8:443",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if !strings.Contains(err.Error(), "multiple dials failed") {
		t.Fatal("unexpected error")
	}
	if conn != nil {
		t.Fatal("non-nil connection")
	}
}

func newdialer() *Dialer {
	return New(
		time.Now(),
		handlers.NoHandler,
		new(net.Dialer),
		new(net.Resolver),
		true,
	)
}
