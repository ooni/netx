// httpclient is a simple HTTP command line client.
//
// Usage:
//
//   httpclient -dns-server <URL> -url <URL>
//
//   httpclient -help
//
// The default is to use the system DNS. The -dns-server <URL> flag
// allows to choose what DNS transport to use (see below).
//
// We emit JSONL messages on the stdout showing what we are
// currently doing. We also print the final result on the stdout.
//
// Examples:
//
//   ./httpclient -dns-server system:/// -url https://ooni.org/ ...
//   ./httpclient -dns-server https://cloudflare-dns.com/dns-query -url https://ooni.org/ ...
//   ./httpclient -dns-server dot://dns.quad9.net -url https://ooni.org/ ...
//   ./httpclient -dns-server dot://1.1.1.1:853 -url https://ooni.org/ ...
//   ./httpclient -dns-server tcp://8.8.8.8:53 -url https://ooni.org/ ...
//   ./httpclient -dns-server udp://1.1.1.1:53 -url https://ooni.org/ ...
package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/m-lab/go/rtx"
	"github.com/ooni/netx/cmd/common"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/httpx"
)

var (
	flagDNSServer = flag.String("dns-server", "system:///", "Server to use")
	flagSNI       = flag.String("sni", "", "Force specific SNI")
	flagURL       = flag.String("url", "https://ooni.io/", "URL to fetch")
)

func main() {
	flag.Parse()
	err := mainfunc()
	rtx.Must(err, "mainfunc failed")
}

func mainfunc() (err error) {
	defer func() {
		if recover() != nil {
			// JUST KNOW WE ARRIVED HERE
		}
	}()
	client := httpx.NewClient(handlers.StdoutHandler)
	if *common.FlagHelp {
		flag.CommandLine.SetOutput(os.Stdout)
		fmt.Printf("Usage: httpclient [flags] -url <url>\n")
		flag.PrintDefaults()
		fmt.Printf("\nExamples:\n")
		fmt.Printf("%s\n",
			"  ./httpclient system:/// -url https://ooni.org/ ...",
		)
		fmt.Printf("%s\n",
			"  ./httpclient -dns-server https://cloudflare-dns.com/dns-query "+
				"-url https://ooni.org/ ...",
		)
		fmt.Printf("%s\n",
			"  ./httpclient -dns-server dot://dns.quad9.net -url https://ooni.org/ ...",
		)
		fmt.Printf("%s\n",
			"  ./httpclient -dns-server dot://1.1.1.1:853 -url https://ooni.org/ ...",
		)
		fmt.Printf("%s\n",
			"  ./httpclient -dns-server tcp://8.8.8.8:53 -url https://ooni.org/ ...",
		)
		fmt.Printf("%s\n",
			"  ./httpclient -dns-server udp://1.1.1.1:53 -url https://ooni.org/ ...",
		)
		fmt.Printf("\nWe'll select a suitable backend for each transport.\n")
		return nil
	}

	urlDNSServer, err := url.Parse(*flagDNSServer)
	rtx.PanicOnError(err, "-dns-server argument is not a valid URL")

	if urlDNSServer.Scheme == "system" {
		err = client.ConfigureDNS("system", "")
	} else if urlDNSServer.Scheme == "udp" {
		err = client.ConfigureDNS("udp", urlDNSServer.Host)
	} else if urlDNSServer.Scheme == "tcp" {
		err = client.ConfigureDNS("tcp", urlDNSServer.Host)
	} else if urlDNSServer.Scheme == "dot" {
		err = client.ConfigureDNS("dot", urlDNSServer.Host)
	} else if urlDNSServer.Scheme == "https" {
		err = client.ConfigureDNS("doh", urlDNSServer.String())
	} else if *flagDNSServer != "" {
		err = errors.New("invalid -dns-server argument")
	}
	rtx.PanicOnError(err, "cannot configure DNS server")
	err = client.ForceSpecificSNI(*flagSNI)
	rtx.PanicOnError(err, "cannot force specific SNI")
	err = fetch(client.HTTPClient, *flagURL)
	rtx.PanicOnError(err, "cannot fetch specific URL")
	return
}

func fetch(client *http.Client, url string) (err error) {
	defer func() {
		if recover() != nil {
			// JUST KNOW WE ARRIVED HERE
		}
	}()
	resp, err := client.Get(url)
	rtx.PanicOnError(err, "client.Get failed")
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	rtx.PanicOnError(err, "ioutil.ReadAll failed")
	fmt.Printf(
		`{"_HTTPResponseBody": "%s"}`+"\n", base64.StdEncoding.EncodeToString(data),
	)
	return
}
