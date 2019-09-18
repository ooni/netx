package httpx

import (
	"io/ioutil"
	"testing"

	"github.com/bassosimone/netx/internal/testingx"
)

func TestIntegration(t *testing.T) {
	client := NewClient(testingx.StdoutHandler)
	resp, err := client.HTTPClient.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}
