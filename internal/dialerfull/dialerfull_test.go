package dialerfull

import (
	"context"
	"crypto/x509"
	"net"
	"testing"
	"time"
)

func TestIntegrationDialContext(t *testing.T) {
	dialer := NewDialer(time.Now())
	conn, err := dialer.DialContext(
		context.Background(), "tcp", "www.google.com:80",
	)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationDialTLS(t *testing.T) {
	dialer := NewDialer(time.Now())
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com:443",
	)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationInvalidAddress(t *testing.T) {
	dialer := NewDialer(time.Now())
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com", /* no port! */
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationLookupFailure(t *testing.T) {
	dialer := NewDialer(time.Now())
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "antani.ooni.io:443",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationTLSHandshakeSetDeadlineError(t *testing.T) {
	dialer := NewDialer(time.Now())
	dialer.startTLSHandshakeHook = func(c net.Conn) {
		c.Close() // close the connection so SetDealine should fail
	}
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "ooni.io:443",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestSetCABundleExisting(t *testing.T) {
	dialer := NewDialer(time.Now())
	err := dialer.SetCABundle("../../testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetCABundleNonexisting(t *testing.T) {
	dialer := NewDialer(time.Now())
	err := dialer.SetCABundle("../../testdata/cacert-nonexistent.pem")
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestSetCABundleWAI(t *testing.T) {
	dialer := NewDialer(time.Now())
	err := dialer.SetCABundle("../../testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com:443",
	)
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
	dialer := NewDialer(time.Now())
	err := dialer.ForceSpecificSNI("www.facebook.com")
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com:443",
	)
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
