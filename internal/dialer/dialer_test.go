package dialer

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/model"
)

func TestIntegrationNew(t *testing.T) {
	var dialer model.Dialer = New(
		time.Now(),
		handlers.NoHandler,
		new(net.Resolver),
		new(net.Dialer),
	)
	conn, err := dialer.Dial("tcp", "www.kernel.org:80")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn")
	}
	conn.Close()
}

func TestIntegrationNewTLS(t *testing.T) {
	var dialer model.TLSDialer = NewTLS(
		time.Now(),
		handlers.NoHandler,
		new(net.Dialer),
		new(tls.Config),
	)
	conn, err := dialer.DialTLS("tcp", "www.kernel.org:443")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn")
	}
	conn.Close()
}
