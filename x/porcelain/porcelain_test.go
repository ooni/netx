package porcelain

import "testing"

func TestIntegration(t *testing.T) {
	req, err := NewHTTPRequest("GET", "http://www.x.org", nil)
	if err != nil {
		t.Fatal(err)
	}
	root := RequestMeasurementRoot(req)
	if root == nil {
		t.Fatal("unexpected nil root")
	}
}
