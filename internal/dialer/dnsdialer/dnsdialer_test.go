package dnsdialer

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/model"
)

func TestIntegrationDial(t *testing.T) {
	dialer := newdialer()
	conn, err := dialer.Dial("tcp", "www.google.com:80")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationDialAddress(t *testing.T) {
	dialer := newdialer()
	conn, err := dialer.Dial("tcp", "8.8.8.8:853")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationNoPort(t *testing.T) {
	dialer := newdialer()
	conn, err := dialer.Dial("tcp", "antani.ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationLookupFailure(t *testing.T) {
	dialer := newdialer()
	conn, err := dialer.Dial("tcp", "antani.ooni.io:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationDialTCPFailure(t *testing.T) {
	dialer := newdialer()
	// The port is unreachable and filtered. The timeout is here
	// to make sure that we don't run for too much time.
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	conn, err := dialer.DialContext(ctx, "tcp", "ooni.io:12345")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func newdialer() model.Dialer {
	return New(
		time.Now(),
		handlers.NoHandler,
		new(net.Resolver),
		new(net.Dialer),
	)
}
