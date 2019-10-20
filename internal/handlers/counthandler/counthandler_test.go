package counthandler

import (
	"sync"
	"testing"
	"time"

	"github.com/ooni/netx/model"
)

func TestIntegration(t *testing.T) {
	const count = 3
	var (
		handler Handler
		wg      sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		for i := 0; i < count; i++ {
			time.Sleep(250 * time.Millisecond)
			handler.OnMeasurement(model.Measurement{})
		}
		wg.Done()
	}()
	wg.Wait()
	if handler.Count != 3 {
		t.Fatal("did not record all emitted measurements")
	}
}
