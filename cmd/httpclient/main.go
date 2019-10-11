// httpclient is a simple HTTP command line client.
//
// Usage:
//
//   httpclient -dns-transport system|godns|tcp|udp|dot|doh -url <URL>
//
//   httpclient -help
//
// The default is to use the system DNS. Use -dns-engine to force
// a different type of DNS transport. We'll use a good default resolver
// for the selected transport. This only works on Unix.
//
// We emit JSONL messages on the stdout showing what we are
// currently doing. We also print the final result on the stdout.
//
// Examples:
//
//   ./httpclient -dns-server system:/// -url https://ooni.org/ ...
//   ./httpclient -dns-server godns:/// -url https://ooni.org/ ...
//   ./httpclient -dns-server https://cloudflare-dns.com/dns-query -url https://ooni.org/ ...
//   ./httpclient -dns-server dot://dns.quad9.net -url https://ooni.org/ ...
//   ./httpclient -dns-server dot://1.1.1.1:853 -url https://ooni.org/ ...
//   ./httpclient -dns-server tcp://8.8.8.8:53 -url https://ooni.org/ ...
//   ./httpclient -dns-server udp://1.1.1.1:53 -url https://ooni.org/ ...
package main

import (
	"context"
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
	"github.com/ooni/netx/httpx/httptracex"
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
	client := httpx.NewClient(handlers.NoHandler)
	if *common.FlagHelp {
		flag.CommandLine.SetOutput(os.Stdout)
		fmt.Printf("Usage: httpclient [flags] -url <url>\n")
		flag.PrintDefaults()
		fmt.Printf("\nExamples:\n")
		fmt.Printf("%s\n",
			"  ./httpclient system:/// -url https://ooni.org/ ...",
		)
		fmt.Printf("%s\n",
			"  ./httpclient -dns-server godns:/// -url https://ooni.org/ ...",
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
		fmt.Printf("\nWe'll select a suitable backend for each transport. Note\n")
		fmt.Printf("that this only works on Unix.\n")
		return nil
	}

	ctx := context.Background()
	ctx = httptracex.ContextWithHandler(ctx, handlers.StdoutHandler)

	urlDNSServer, err := url.Parse(*flagDNSServer)
	rtx.PanicOnError(err, "-dns-server argument is not a valid URL")

	if urlDNSServer.Scheme == "system" {
		err = client.ConfigureDNS("system", "")
	} else if urlDNSServer.Scheme == "godns" {
		err = client.ConfigureDNS("godns", "")
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
	err = fetch(ctx, client.HTTPClient, *flagURL)
	rtx.PanicOnError(err, "cannot fetch specific URL")
	return
}

func fetch(ctx context.Context, client *http.Client, url string) (err error) {
	defer func() {
		if recover() != nil {
			// JUST KNOW WE ARRIVED HERE
		}
	}()
	req, err := http.NewRequest("GET", url, nil)
	rtx.PanicOnError(err, "http.NewRequest failed")
	resp, err := client.Do(req.WithContext(ctx))
	rtx.PanicOnError(err, "client.Do failed")
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	rtx.PanicOnError(err, "ioutil.ReadAll failed")
	return
}
