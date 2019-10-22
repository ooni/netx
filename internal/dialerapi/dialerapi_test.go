package dialerapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/tracing"
)

func TestIntegrationDial(t *testing.T) {
	dialer := NewDialer()
	ctx := tracing.WithInfo(context.Background(), tracing.NewInfo(
		"dialerapi_test.go", time.Now(), handlers.NoHandler,
	))
	conn, err := dialer.DialContext(ctx, "tcp", "www.google.com:80")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationDialTLS(t *testing.T) {
	dialer := NewDialer()
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationInvalidAddress(t *testing.T) {
	dialer := NewDialer()
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationFlexibleDialIPAddress(t *testing.T) {
	dialer := NewDialer()
	conn, err := dialer.flexibleDial(
		context.Background(), "tcp", "8.8.8.8:443", true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected a non-nil conn here")
	}
	conn.Close()
}

func TestIntegrationUnexpectedDomain(t *testing.T) {
	dialer := NewDialer()
	conn, err := dialer.flexibleDial(
		context.Background(), "tcp", "www.google.com:443", true,
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationLookupFailure(t *testing.T) {
	dialer := NewDialer()
	conn, err := dialer.flexibleDial(
		context.Background(), "tcp", "antani.ooni.io:443", false,
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationDialTCPFailure(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dialer := NewDialer()
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
	dialer := NewDialer()
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

func TestIntegrationDialInvalidSNI(t *testing.T) {
	dialer := NewDialer()
	dialer.TLSConfig = &tls.Config{
		ServerName: "www.google.com",
	}
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "ooni.io:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestSetCABundleExisting(t *testing.T) {
	dialer := NewDialer()
	err := dialer.SetCABundle("../../testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetCABundleNonexisting(t *testing.T) {
	dialer := NewDialer()
	err := dialer.SetCABundle("../../testdata/cacert-nonexistent.pem")
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestSetCABundleWAI(t *testing.T) {
	dialer := NewDialer()
	err := dialer.SetCABundle("../../testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if _, ok := err.(x509.UnknownAuthorityError); !ok {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestForceSpecificSNI(t *testing.T) {
	dialer := NewDialer()
	err := dialer.ForceSpecificSNI("www.facebook.com")
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if _, ok := err.(x509.HostnameError); !ok {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected a nil connection here")
	}
}

func TestFlexibleDialSplitHostPort(t *testing.T) {
	dialer := NewDialer()
	conn, err := dialer.flexibleDial(context.Background(), "tcp", "antani!", false)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestDialTLSContextFlexibleDialError(t *testing.T) {
	dialer := NewDialer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // should then fail immediately
	conn, err := dialer.DialTLSContext(ctx, "tcp", "www.google.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}
