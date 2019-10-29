package porcelain

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
)

func TestIntegration(t *testing.T) {
	body := strings.NewReader("antani")
	req, err := NewHTTPRequest("POST", "http://www.x.org", body)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != "POST" {
		t.Fatal("unexpected method")
	}
	if req.URL.Scheme != "http" {
		t.Fatal("unexpected scheme")
	}
	if req.URL.Host != "www.x.org" {
		t.Fatal("unexpected host")
	}
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte("antani")) {
		t.Fatal("unexpected body")
	}
	root := RequestMeasurementRoot(req)
	if root == nil {
		t.Fatal("unexpected nil root")
	}
}
