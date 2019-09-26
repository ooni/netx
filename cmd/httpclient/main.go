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
//   ./httpclient -dns-transport system ...
//   ./httpclient -dns-transport godns ...
//   ./httpclient -dns-transport doh ...
//   ./httpclient -dns-transport dot ...
//   ./httpclient -dns-transport tcp ...
//   ./httpclient -dns-transport udp ...
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/m-lab/go/rtx"
	"github.com/ooni/netx/cmd/common"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/httpx"
)

var (
	flagDNSTransport = flag.String("dns-transport", "", "DNS transport to use")
	flagURL          = flag.String("url", "https://ooni.io/", "URL to fetch")
)

func main() {
	flag.Parse()
	err := mainfunc()
	rtx.Must(err, "mainfunc failed")
}

func mainfunc() (err error) {
	client := httpx.NewClient(handlers.StdoutHandler)
	if *common.FlagHelp {
		flag.CommandLine.SetOutput(os.Stdout)
		fmt.Printf("Usage: dnsclient [flags]\n")
		flag.PrintDefaults()
		fmt.Printf("\nExamples:\n")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport system ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport godns ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport doh ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport dot ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport tcp ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport udp ...")
		fmt.Printf("\nWe'll select a suitable backend for each transport. Note\n")
		fmt.Printf("that this only works on Unix.\n")
		return nil
	}
	if *flagDNSTransport == "system" {
		err = client.ConfigureDNS("system", "")
	} else if *flagDNSTransport == "godns" {
		err = client.ConfigureDNS("godns", "")
	} else if *flagDNSTransport == "udp" {
		err = client.ConfigureDNS("udp", "1.1.1.1:53")
	} else if *flagDNSTransport == "tcp" {
		err = client.ConfigureDNS("tcp", "8.8.8.8:53")
	} else if *flagDNSTransport == "dot" {
		err = client.ConfigureDNS("dot", "dns.quad9.net")
	} else if *flagDNSTransport == "doh" {
		err = client.ConfigureDNS("doh", "https://cloudflare-dns.com/dns-query")
	} else if *flagDNSTransport != "" {
		err = errors.New("invalid -dns-transport argument")
	}
	if err == nil {
		fetch(client.HTTPClient, *flagURL)
	}
	return
}

func fetch(client *http.Client, url string) {
	resp, err := client.Get(url)
	if err == nil {
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
}
