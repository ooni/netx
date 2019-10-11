package tracing_test

import (
	"context"
	"testing"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/tracing"
	"github.com/ooni/netx/model"
)

type handler struct {
	m model.Measurement
}

func (h *handler) OnMeasurement(m model.Measurement) {
	h.m = m
}

func TestIntegration(t *testing.T) {
	ctx := context.Background()
	h1 := &handler{}
	ctx = tracing.WithHandler(ctx, h1)
	h2 := &handler{}
	ctx = tracing.WithHandler(ctx, h2)
	h3 := &handler{}
	ctx = tracing.WithHandler(ctx, h3)
	handler := tracing.ContextHandler(ctx)
	if handler == nil {
		t.Fatal("handler is nil")
	}
	handler.OnMeasurement(model.Measurement{
		Resolve: &model.ResolveEvent{
			ConnID: 17,
		},
	})
	if h1.m.Resolve.ConnID != 17 {
		t.Fatal("not written on first handler")
	}
	if h2.m.Resolve.ConnID != 17 {
		t.Fatal("not written on second handler")
	}
	if h3.m.Resolve.ConnID != 17 {
		t.Fatal("not written on third handler")
	}
}

func TestWithHandlerFailure(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("we expected to panic here")
		}
	}()
	ctx := context.Background()
	tracing.WithHandler(ctx, nil)
}

func TestContextHandlerDefault(t *testing.T) {
	ctx := context.Background()
	handler := tracing.ContextHandler(ctx)
	if handler == nil {
		t.Fatal("handler is nil")
	}
	if handler != handlers.NoHandler {
		t.Fatal("handler is not NoHandler")
	}
}

func TestCompose(t *testing.T) {
	h1 := &handler{}
	h2 := &handler{}
	h3 := tracing.Compose(h1, h2)
	h4 := &handler{}
	h5 := tracing.Compose(h4, h3)
	h5.OnMeasurement(model.Measurement{
		Resolve: &model.ResolveEvent{
			ConnID: 17,
		},
	})
	if h1.m.Resolve.ConnID != 17 {
		t.Fatal("not written on first handler")
	}
	if h2.m.Resolve.ConnID != 17 {
		t.Fatal("not written on second handler")
	}
	if h4.m.Resolve.ConnID != 17 {
		t.Fatal("not written on third handler")
	}
}
