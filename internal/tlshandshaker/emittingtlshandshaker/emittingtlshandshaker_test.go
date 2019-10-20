package emittingtlshandshaker

import (
	"context"
	"crypto/tls"
	"net"
	"testing"

	"github.com/ooni/netx/internal/handlers/counthandler"
	"github.com/ooni/netx/internal/tlshandshaker/ootlshandshaker"
	"github.com/ooni/netx/internal/tracing"
)

func TestIntegrationSuccess(t *testing.T) {
	info := &tracing.Info{
		Handler: &counthandler.Handler{},
	}
	ctx := tracing.WithInfo(context.Background(), info)
	handshaker := New(ootlshandshaker.New())
	conn, err := (&net.Dialer{}).Dial("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	tlsconn, err := handshaker.Do(
		ctx, conn, &tls.Config{}, "youtube.com", // 🙃
	)
	if err != nil {
		t.Fatal(err)
	}
	if tlsconn == nil {
		t.Fatal("expected non-nil tslconn")
	}
	tlsconn.Close()
	if info.Handler.(*counthandler.Handler).Count < 0 {
		t.Fatal("no measurements saved")
	}
}

func TestIntegrationTLSHandshakeFailure(t *testing.T) {
	info := &tracing.Info{
		Handler: &counthandler.Handler{},
	}
	ctx := tracing.WithInfo(context.Background(), info)
	handshaker := New(ootlshandshaker.New())
	conn, err := (&net.Dialer{}).Dial("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	tlsconn, err := handshaker.Do(
		ctx, conn, &tls.Config{}, "x.org",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if tlsconn == nil {
		t.Fatal("expected non-nil tslconn")
	}
	tlsconn.Close()
	if info.Handler.(*counthandler.Handler).Count < 0 {
		t.Fatal("no measurements saved")
	}
}

func TestIntegrationContextDeadline(t *testing.T) {
	info := &tracing.Info{
		Handler: &counthandler.Handler{},
	}
	ctx := tracing.WithInfo(context.Background(), info)
	handshaker := New(ootlshandshaker.New())
	conn, err := (&net.Dialer{}).Dial("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel() // fail now
	tlsconn, err := handshaker.Do(
		ctx, conn, &tls.Config{}, "x.org",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if tlsconn != nil {
		t.Fatal("expected nil tslconn")
	}
	if info.Handler.(*counthandler.Handler).Count != 0 {
		t.Fatal("measurements saved")
	}
}
