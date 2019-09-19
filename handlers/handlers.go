// Package handlers contains default model.Handler handlers.
package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/bassosimone/netx/model"
	"github.com/m-lab/go/rtx"
)

type handler struct{}

func (handler) OnMeasurement(m model.Measurement) {
	data, err := json.Marshal(m)
	rtx.Must(err, "unexpected json.Marshal failure")
	fmt.Printf("%s\n", string(data))
}

// StdoutHandler is a Handler that logs on stdout.
var StdoutHandler handler
