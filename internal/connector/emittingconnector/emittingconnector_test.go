package emittingconnector

import (
	"context"
	"testing"
	"time"

	"github.com/ooni/netx/internal/connector/ooconnector"
	"github.com/ooni/netx/internal/handlers/counthandler"
	"github.com/ooni/netx/internal/tracing"
)

func TestIntegration(t *testing.T) {
	info := tracing.NewInfo(
		"emttingconnector_test.go", time.Now(),
		&counthandler.Handler{},
	)
	ctx := tracing.WithInfo(context.Background(), info)
	connector := New(ooconnector.New())
	conn, err := connector.DialContext(ctx, "tcp", "8.8.8.8:53")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected a nil conn")
	}
	conn.Close()
	if info.Handler.(*counthandler.Handler).Count < 0 {
		t.Fatal("no measurements saved")
	}
}
