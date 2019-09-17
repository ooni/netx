package dialerapi

import (
	"testing"
	"time"

	"github.com/bassosimone/netx/internal/testingx"
	"github.com/bassosimone/netx/model"
)

func TestIntegrationDial(t *testing.T) {
	ch := make(chan model.Measurement)
	cancel := testingx.SpawnLogger(ch)
	defer cancel()
	dialer := NewDialer(time.Now(), ch)
	conn, err := dialer.Dial("tcp", "www.google.com:80")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationDialTLS(t *testing.T) {
	ch := make(chan model.Measurement)
	cancel := testingx.SpawnLogger(ch)
	defer cancel()
	dialer := NewDialer(time.Now(), ch)
	conn, err := dialer.DialTLS("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}
