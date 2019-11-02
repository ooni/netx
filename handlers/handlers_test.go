package handlers_test

import (
	"testing"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/modelx"
)

func TestIntegration(t *testing.T) {
	handlers.NoHandler.OnMeasurement(modelx.Measurement{})
	handlers.StdoutHandler.OnMeasurement(modelx.Measurement{})
}
