package savinghandler

import (
	"sync"
	"testing"

	"github.com/ooni/netx/model"
)

func TestIntegration(t *testing.T) {
	var (
		wg      sync.WaitGroup
		handler Handler
	)
	wg.Add(1)
	go func() {
		handler.OnMeasurement(model.Measurement{
			HTTPConnectionReady: &model.HTTPConnectionReadyEvent{
				Time:          4,
				TransactionID: 155,
			},
		})
		wg.Done()
	}()
	wg.Wait()
	if len(handler.All) != 1 {
		t.Fatal("measurements not saved")
	}
	if handler.All[0].HTTPConnectionReady == nil {
		t.Fatal("specific event is missing")
	}
	evt := handler.All[0].HTTPConnectionReady
	if evt.Time != 4 || evt.TransactionID != 155 {
		t.Fatal("specific event is corrupt")
	}
}
