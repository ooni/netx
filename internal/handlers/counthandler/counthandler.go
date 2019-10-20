// Package counthandler contains a handler that counts
package counthandler

import (
	"sync/atomic"

	"github.com/ooni/netx/model"
)

// Handler is the count handler
type Handler struct {
	Count int64
}

// OnMeasurement counts the number of emitted measurements
func (h *Handler) OnMeasurement(m model.Measurement) {
	atomic.AddInt64(&h.Count, 1)
}
