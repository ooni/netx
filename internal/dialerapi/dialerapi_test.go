package dialerapi_test

import (
	"context"
	"testing"
	"time"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/testingx"
)

func TestIntegrationDial(t *testing.T) {
	dialer := dialerapi.NewDialer(time.Now(), testingx.StdoutHandler)
	conn, err := dialer.Dial("tcp", "www.google.com:80")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationDialTLS(t *testing.T) {
	dialer := dialerapi.NewDialer(time.Now(), testingx.StdoutHandler)
	conn, err := dialer.DialTLS("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationInvalidAddress(t *testing.T) {
	dialer := dialerapi.NewDialer(time.Now(), testingx.StdoutHandler)
	conn, err := dialer.DialTLS("tcp", "www.google.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationUnexpectedDomain(t *testing.T) {
	dialer := dialerapi.NewDialer(time.Now(), testingx.StdoutHandler)
	conn, onlyhost, onlyport, err := dialer.DialContextEx(
		context.Background(), "tcp", "www.google.com:443", true,
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
	dialer := dialerapi.NewDialer(time.Now(), testingx.StdoutHandler)
	conn, onlyhost, onlyport, err := dialer.DialContextEx(
		context.Background(), "tcp", "antani.ooni.io:443", false,
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

func TestDialTCPFailure(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dialer := dialerapi.NewDialer(time.Now(), testingx.StdoutHandler)
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

func TestDialDNSFailure(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dialer := dialerapi.NewDialer(time.Now(), testingx.StdoutHandler)
	// The insane timeout is such that the DNS resolver fails because it
	// times out when trying to dial for the default server. (This is
	// a test that only makes sense on Unix.)
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()
	conn, err := dialer.DialContext(ctx, "tcp", "ooni.io:80")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}
