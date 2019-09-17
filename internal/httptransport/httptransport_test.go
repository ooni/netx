package httptransport

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/bassosimone/netx/internal/testingx"
	"github.com/bassosimone/netx/model"
)

func TestIntegration(t *testing.T) {
	ch := make(chan model.Measurement)
	client := &http.Client{
		Transport: NewTransport(time.Now(), ch),
	}
	cancel := testingx.SpawnLogger(ch)
	defer cancel()
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}
