package dnsconf_test

import (
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/dialer"
	"github.com/ooni/netx/internal/dnsconf"
)

func testconfigurednsquick(t *testing.T, network, address string) {
	d := dialer.NewDialer(time.Now(), handlers.NoHandler)
	err := dnsconf.ConfigureDNS(d, network, address)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := d.DialTLS("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn here")
	}
	conn.Close()
}

func TestIntegrationConfigureSystemDNS(t *testing.T) {
	testconfigurednsquick(t, "system", "")
}
