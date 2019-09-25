package handlers_test

import (
	"testing"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/model"
)

func TestIntegration(t *testing.T) {
	handlers.NoHandler.OnMeasurement(model.Measurement{})
	handlers.StdoutHandler.OnMeasurement(model.Measurement{})
}
