package httptransport

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/bassosimone/netx/internal/testingx"
)

func TestIntegration(t *testing.T) {
	client := &http.Client{
		Transport: NewTransport(time.Now(), testingx.StdoutHandler),
	}
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
