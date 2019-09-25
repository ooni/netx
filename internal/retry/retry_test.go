package retry_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/netx/internal/retry"
)

func TestRetryFailure(t *testing.T) {
	err := retry.Retry(context.Background(), func() error {
		return errors.New("mocked error")
	})
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestRetrySuccess(t *testing.T) {
	err := retry.Retry(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRetryTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	err := retry.Retry(ctx, func() error {
		return errors.New("mocked error")
	})
	if err == nil {
		t.Fatal("expected an error here")
	}
}