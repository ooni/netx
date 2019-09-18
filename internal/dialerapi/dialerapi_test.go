package dialerapi

import (
	"testing"
	"time"

	"github.com/bassosimone/netx/internal/testingx"
)

func TestIntegrationDial(t *testing.T) {
	dialer := NewDialer(time.Now(), testingx.StdoutHandler)
	conn, err := dialer.Dial("tcp", "www.google.com:80")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationDialTLS(t *testing.T) {
	dialer := NewDialer(time.Now(), testingx.StdoutHandler)
	conn, err := dialer.DialTLS("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}
