// Package handlers contains default model.Handler handlers.
package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/m-lab/go/rtx"
	"github.com/ooni/netx/model"
)

type stdoutHandler struct{}

func (stdoutHandler) OnMeasurement(m model.Measurement) {
	data, err := json.Marshal(m)
	rtx.Must(err, "unexpected json.Marshal failure")
	fmt.Printf("%s\n", string(data))
}

// StdoutHandler is a Handler that logs on stdout.
var StdoutHandler stdoutHandler

type noHandler struct{}

func (noHandler) OnMeasurement(m model.Measurement) {
}

// NoHandler is a Handler that does not print anything
var NoHandler noHandler
