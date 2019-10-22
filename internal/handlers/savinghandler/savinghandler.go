// Package savinghandler contains a handler that saves measurements
package savinghandler

import (
	"sync"

	"github.com/ooni/netx/model"
)

// Handler is a handler that saves measurements
type Handler struct {
	All []model.Measurement
	mu  sync.Mutex
}

// OnMeasurement counts the number of emitted measurements
func (h *Handler) OnMeasurement(m model.Measurement) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.All = append(h.All, m)
}
