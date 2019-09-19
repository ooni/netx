package httpx_test

import (
	"io/ioutil"
	"testing"

	"github.com/bassosimone/netx/handlers"
	"github.com/bassosimone/netx/httpx"
)

func TestIntegration(t *testing.T) {
	client := httpx.NewClient(handlers.StdoutHandler)
	defer client.Transport.CloseIdleConnections()
	err := client.ConfigureDNS("udp", "1.1.1.1:53")
	if err != nil {
		t.Fatal(err)
	}
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
