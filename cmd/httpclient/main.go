// httpclient is a simple HTTP command line client. It will fetch all the
// URLs passed to the command line. While doing that, it will log http and
// network level events on the standard output in JSONL format.
package main

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/bassosimone/netx/httpx"
	"github.com/bassosimone/netx/internal/testingx"
)

func fetch(client *http.Client, url string) {
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	ioutil.ReadAll(resp.Body)
	resp.Body.Close()
}

func main() {
	client := httpx.NewClient(testingx.StdoutHandler)
	var err error
	//err = client.ConfigureDNS("udp", "1.1.1.1:53")
	//err = client.ConfigureDNS("tcp", "8.8.8.8:53")
	//err = client.ConfigureDNS("dot", "dns.quad9.net")
	//err = client.ConfigureDNS("doh", "https://cloudflare-dns.com/dns-query")
	if err != nil {
		panic(err)
	}
	for _, url := range os.Args[1:] {
		fetch(client.HTTPClient, url)
	}
}
