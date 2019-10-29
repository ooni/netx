// Package porcelain contains useful high level functionality
package porcelain

import (
	"io"
	"net/http"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/model"
)

// NewHTTPRequest is like http.NewRequest except that it also adds
// to such request a configured MeasurementRoot. The configured
// MeasurementRoot will have all experimental extensions enabled.
func NewHTTPRequest(method, URL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, URL, body)
	if err == nil {
		root := &model.MeasurementRoot{
			Beginning: time.Now(),
			Handler:   handlers.NoHandler,
		}
		ctx := model.WithMeasurementRoot(req.Context(), root)
		req = req.WithContext(ctx)
	}
	return req, err
}

// RequestMeasurementRoot returns the MeasurementRoot that has been
// configured into the context of a request, or nil.
func RequestMeasurementRoot(req *http.Request) *model.MeasurementRoot {
	return model.ContextMeasurementRoot(req.Context())
}
