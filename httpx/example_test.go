package httpx_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/bassosimone/netx/httpx"
)

func Example() {
	getfunc := func(client *http.Client, url string) error {
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		_, err = ioutil.ReadAll(resp.Body)
		return err
	}
	client := httpx.NewClient()
	err := getfunc(client.HTTPClient, "http://facebook.com")
	if err != nil {
		log.Fatal(err)
	}
	netMeasurements := client.PopNetMeasurements()
	httpMeasurements := client.PopHTTPMeasurements()
	fmt.Printf("%+v %+v\n", len(netMeasurements) > 0, len(httpMeasurements) > 0)
	// Output: true true
}
