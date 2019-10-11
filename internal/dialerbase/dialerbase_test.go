package dialerbase_test

import (
	"context"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/dialerbase"
)

func TestIntegrationSuccess(t *testing.T) {
	dialer := dialerbase.NewDialer(
		time.Now(),
	)
	conn, err := dialer.DialHostPort(
		context.Background(), handlers.NoHandler, "tcp", "8.8.8.8", "53", 17,
	)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationErrorDomain(t *testing.T) {
	dialer := dialerbase.NewDialer(
		time.Now(),
	)
	conn, err := dialer.DialHostPort(
		context.Background(), handlers.NoHandler, "tcp", "dns.google.com", "53", 17,
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestIntegrationErrorNoConnect(t *testing.T) {
	dialer := dialerbase.NewDialer(
		time.Now(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()
	conn, err := dialer.DialHostPort(
		ctx, handlers.NoHandler, "tcp", "8.8.8.8", "53", 17,
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if ctx.Err() == nil {
		t.Fatal("expected context to be expired here")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}
