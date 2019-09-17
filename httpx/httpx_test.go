package httpx

import (
	"io/ioutil"
	"testing"

	"github.com/bassosimone/netx/internal/testingx"
	"github.com/bassosimone/netx/model"
)

func TestIntegration(t *testing.T) {
	ch := make(chan model.Measurement)
	client := NewClient(ch)
	cancel := testingx.SpawnLogger(ch)
	defer cancel()
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
