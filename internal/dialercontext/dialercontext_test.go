package dialercontext_test

import (
	"context"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/dialercontext"
)

func TestIntegrationDialContext(t *testing.T) {
	dialer := dialercontext.NewDialer(time.Now())
	conn, err := dialer.DialContext(
		context.Background(), "tcp", "www.google.com:80",
	)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationInvalidAddress(t *testing.T) {
	dialer := dialercontext.NewDialer(time.Now())
	conn, err := dialer.DialContext(
		context.Background(), "tcp", "www.google.com", // missing port
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationDialContextExIPAddress(t *testing.T) {
	dialer := dialercontext.NewDialer(time.Now())
	conn, onlyhost, onlyport, err := dialer.DialContextEx(
		context.Background(), handlers.NoHandler, "tcp", "8.8.8.8:443", true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if onlyhost != "8.8.8.8" {
		t.Fatal("unexpected onlyhost value")
	}
	if onlyport != "443" {
		t.Fatal("unexpected onlyport value")
	}
	if conn == nil {
		t.Fatal("expected a non-nil conn here")
	}
	conn.Close()
}

func TestIntegrationUnexpectedDomain(t *testing.T) {
	dialer := dialercontext.NewDialer(time.Now())
	conn, onlyhost, onlyport, err := dialer.DialContextEx(
		context.Background(), handlers.NoHandler, "tcp", "www.google.com:443", true,
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if onlyhost != "www.google.com" {
		t.Fatal("unexpected onlyhost value")
	}
	if onlyport != "443" {
		t.Fatal("unexpected onlyport value")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationLookupFailure(t *testing.T) {
	dialer := dialercontext.NewDialer(time.Now())
	conn, onlyhost, onlyport, err := dialer.DialContextEx(
		context.Background(), handlers.NoHandler,
		"tcp", "antani.ooni.io:443", false,
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if onlyhost != "antani.ooni.io" {
		t.Fatal("unexpected onlyhost value")
	}
	if onlyport != "443" {
		t.Fatal("unexpected onlyport value")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationDialTCPFailure(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dialer := dialercontext.NewDialer(time.Now())
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
