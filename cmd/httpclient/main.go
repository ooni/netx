// httpclient is a simple HTTP command line client.
//
// Usage:
//
//   dnsclient -dns-transport tcp|udp|dot|doh
//             -url <URL>
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
//   ./httpclient -dns-transport doh ...
//   ./httpclient -dns-transport dot ...
//   ./httpclient -dns-transport tcp ...
//   ./httpclient -dns-transport udp ...
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/httpx"
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
	client := httpx.NewClient(handlers.StdoutHandler)
	var (
		err              error
		flagDNSTransport = flag.String("dns-transport", "", "DNS transport to use")
		flagHelp         = flag.Bool("h", false, "Print help message")
		flagURL          = flag.String("url", "https://ooni.io/", "URL to fetch")
	)
	flag.Parse()
	if *flagHelp {
		flag.CommandLine.SetOutput(os.Stdout)
		fmt.Printf("Usage: dnsclient [flags]\n")
		flag.PrintDefaults()
		fmt.Printf("\nExamples:\n")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport doh ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport dot ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport tcp ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport udp ...")
		fmt.Printf("\nWe'll select a suitable backend for each transport. Note\n")
		fmt.Printf("that this only works on Unix.\n")
		os.Exit(0)
	}
	if *flagDNSTransport == "udp" {
		err = client.ConfigureDNS("udp", "1.1.1.1:53")
	} else if *flagDNSTransport == "tcp" {
		err = client.ConfigureDNS("tcp", "8.8.8.8:53")
	} else if *flagDNSTransport == "dot" {
		err = client.ConfigureDNS("dot", "dns.quad9.net")
	} else if *flagDNSTransport == "doh" {
		err = client.ConfigureDNS("doh", "https://cloudflare-dns.com/dns-query")
	} else if *flagDNSTransport != "" {
		log.Fatal("invalid -dns-transport argument")
	}
	if err != nil {
		log.Fatal(err)
	}
	fetch(client.HTTPClient, *flagURL)
}
