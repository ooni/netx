package netx_test

import (
	"context"
	"crypto/x509"
	"testing"

	"github.com/ooni/netx"
	"github.com/ooni/netx/handlers"
)

func TestIntegrationDialer(t *testing.T) {
	dialer := netx.NewDialer(handlers.NoHandler)
	err := dialer.ConfigureDNS("udp", "1.1.1.1:53")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := dialer.Dial("tcp", "www.google.com:80")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	conn, err = dialer.DialContext(
		context.Background(), "tcp", "www.google.com:80",
	)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	conn, err = dialer.DialTLS("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationResolver(t *testing.T) {
	dialer := netx.NewDialer(handlers.NoHandler)
	resolver, err := dialer.NewResolver("tcp", "1.1.1.1:53")
	if err != nil {
		t.Fatal(err)
	}
	addrs, err := resolver.LookupHost(context.Background(), "ooni.io")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) < 1 {
		t.Fatal("No addresses returned")
	}
}

func TestSetCABundle(t *testing.T) {
	dialer := netx.NewDialer(handlers.NoHandler)
	err := dialer.SetCABundle("testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
}

func TestForceSpecificSNI(t *testing.T) {
	dialer := netx.NewDialer(handlers.NoHandler)
	err := dialer.ForceSpecificSNI("www.facebook.com")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := dialer.DialTLS("tcp", "www.google.com:443")
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
