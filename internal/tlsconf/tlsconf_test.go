package tlsconf

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
)

func TestSetCABundleExisting(t *testing.T) {
	config := &tls.Config{}
	// This CA bundle cannot validate google.com by design, so we would
	// except to not being able to establish a secure connection
	err := SetCABundle(config, "../../testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := tls.Dial("tcp", "www.google.com:443", config)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if _, ok := err.(x509.UnknownAuthorityError); !ok {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestSetCABundleNonexisting(t *testing.T) {
	config := &tls.Config{}
	err := SetCABundle(config, "../../testdata/cacert-nonexistent.pem")
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestForceSpecificSNI(t *testing.T) {
	config := &tls.Config{}
	err := ForceSpecificSNI(config, "www.facebook.com")
	conn, err := tls.Dial("tcp", "www.google.com:443", config)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if _, ok := err.(x509.HostnameError); !ok {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}
