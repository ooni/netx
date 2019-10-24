package dialer

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
)

func TestIntegrationDial(t *testing.T) {
	dialer := NewDialer(time.Now(), handlers.NoHandler)
	conn, err := dialer.Dial("tcp", "www.google.com:80")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationDialTLS(t *testing.T) {
	dialer := NewDialer(time.Now(), handlers.NoHandler)
	conn, err := dialer.DialTLS("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationInvalidAddress(t *testing.T) {
	dialer := NewDialer(time.Now(), handlers.NoHandler)
	conn, err := dialer.Dial("tcp", "www.google.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationInvalidAddressTLS(t *testing.T) {
	dialer := NewDialer(time.Now(), handlers.NoHandler)
	conn, err := dialer.DialTLS("tcp", "www.google.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestIntegrationDialInvalidSNI(t *testing.T) {
	dialer := NewDialer(time.Now(), handlers.NoHandler)
	dialer.TLSConfig = &tls.Config{
		ServerName: "www.google.com",
	}
	conn, err := dialer.DialTLS("tcp", "ooni.io:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestSetCABundleExisting(t *testing.T) {
	dialer := NewDialer(time.Now(), handlers.NoHandler)
	err := dialer.SetCABundle("../../testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetCABundleNonexisting(t *testing.T) {
	dialer := NewDialer(time.Now(), handlers.NoHandler)
	err := dialer.SetCABundle("../../testdata/cacert-nonexistent.pem")
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestSetCABundleWAI(t *testing.T) {
	dialer := NewDialer(time.Now(), handlers.NoHandler)
	err := dialer.SetCABundle("../../testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := dialer.DialTLS("tcp", "www.google.com:443")
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
	dialer := NewDialer(time.Now(), handlers.NoHandler)
	err := dialer.ForceSpecificSNI("www.facebook.com")
	conn, err := dialer.DialTLS("tcp", "www.google.com:443")
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
