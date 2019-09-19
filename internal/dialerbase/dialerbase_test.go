package dialerbase_test

import (
	"context"
	"testing"

	"github.com/bassosimone/netx/internal/dialerbase"
	"github.com/bassosimone/netx/handlers"
)

func TestIntegrationSuccess(t *testing.T) {
	dialer := dialerbase.Dialer{
		Handler: handlers.StdoutHandler,
	}
	conn, err := dialer.DialHostPort(
		context.Background(), "tcp", "8.8.8.8", "53", 17,
	)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestIntegrationErrorDomain(t *testing.T) {
	dialer := dialerbase.Dialer{
		Handler: handlers.StdoutHandler,
	}
	conn, err := dialer.DialHostPort(
		context.Background(), "tcp", "dns.google.com", "53", 17,
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestIntegrationErrorNoConnect(t *testing.T) {
	dialer := dialerbase.Dialer{
		Handler: handlers.StdoutHandler,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()
	conn, err := dialer.DialHostPort(
		ctx, "tcp", "8.8.8.8", "53", 17,
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
