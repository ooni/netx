// Package nohandler implements a do-nothing handler
package nohandler

import "github.com/bassosimone/netx/model"

// S is a nohandler instance
type S struct{}

// OnMeasurement does nothing with the provided measurement.
func (S) OnMeasurement(m model.Measurement) {
}
