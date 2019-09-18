package testingx_test

import (
	"testing"

	"github.com/bassosimone/netx/internal/testingx"
	"github.com/bassosimone/netx/model"
)

func TestIntegration(t *testing.T) {
	testingx.StdoutHandler.OnMeasurement(model.Measurement{})
}
