package netx

import (
	"context"
	"crypto/x509"
	"net"
	"testing"
	"time"

	"github.com/ooni/netx/dnsx"
	"github.com/ooni/netx/handlers"
)

func TestIntegrationDialer(t *testing.T) {
	dialer := NewDialer(handlers.NoHandler)
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
	dialer := NewDialer(handlers.NoHandler)
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
	dialer := NewDialer(handlers.NoHandler)
	err := dialer.SetCABundle("testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
}

func TestForceSpecificSNI(t *testing.T) {
	dialer := NewDialer(handlers.NoHandler)
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

func newresolverwrapper() dnsx.Client {
	return &resolverWrapper{
		beginning: time.Now(),
		handler:   handlers.NoHandler,
		resolver:  &net.Resolver{},
	}
}

func TestResolverWrapperLookupAddr(t *testing.T) {
	resolver := newresolverwrapper()
	names, err := resolver.LookupAddr(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	if names == nil {
		t.Fatal("result is nil")
	}
}

func TestResolverWrapperLookupCNAME(t *testing.T) {
	resolver := newresolverwrapper()
	cname, err := resolver.LookupCNAME(context.Background(), "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if cname == "" {
		t.Fatal("result is empty string")
	}
}

func TestResolverWrapperLookupHost(t *testing.T) {
	resolver := newresolverwrapper()
	addrs, err := resolver.LookupHost(context.Background(), "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if addrs == nil {
		t.Fatal("result is nil")
	}
}

func TestResolverWrapperLookupMX(t *testing.T) {
	resolver := newresolverwrapper()
	records, err := resolver.LookupMX(context.Background(), "google.com")
	if err != nil {
		t.Fatal(err)
	}
	if records == nil {
		t.Fatal("result is nil")
	}
}

func TestResolverWrapperLookupNS(t *testing.T) {
	resolver := newresolverwrapper()
	records, err := resolver.LookupNS(context.Background(), "google.com")
	if err != nil {
		t.Fatal(err)
	}
	if records == nil {
		t.Fatal("result is nil")
	}
}
