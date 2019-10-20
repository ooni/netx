package tracing

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ooni/netx/internal/handlers/counthandler"
	"github.com/ooni/netx/model"
)

func TestIntegrationWorks(t *testing.T) {
	const count = 3
	var wg sync.WaitGroup
	wg.Add(1)
	ctx := WithInfo(context.Background(), &Info{
		Handler: &counthandler.Handler{},
	})
	go func(ctx context.Context) {
		info := ContextInfo(ctx)
		for i := 0; i < count; i++ {
			time.Sleep(250 * time.Millisecond)
			info.Handler.OnMeasurement(model.Measurement{})
		}
		wg.Done()
	}(ctx)
	wg.Wait()
	if ContextInfo(ctx).Handler.(*counthandler.Handler).Count != 3 {
		t.Fatal("did not record all emitted measurements")
	}
}

func TestPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	WithInfo(context.Background(), nil)
}
