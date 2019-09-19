package handlers_test

import (
	"testing"

	"github.com/bassosimone/netx/handlers"
	"github.com/bassosimone/netx/model"
)

func TestIntegration(t *testing.T) {
	handlers.NoHandler.OnMeasurement(model.Measurement{})
	handlers.NoHandler.OnMeasurement(model.Measurement{})
}
