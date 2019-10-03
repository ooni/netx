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
//   ./httpclient -dns-transport udp [-dns-udp-server <addr>:<port>] ...
package main

import (
	"encoding/base64"
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
	flagDNSUDPServer = flag.String(
		"dns-udp-server", "1.1.1.1:53", "Server to use with -dns-transport udp",
	)
	flagDNSTransport = flag.String("dns-transport", "", "DNS transport to use")
	flagSNI          = flag.String("sni", "", "Force specific SNI")
	flagURL          = flag.String("url", "https://ooni.io/", "URL to fetch")
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
		fmt.Printf("Usage: httpclient [flags]\n")
		flag.PrintDefaults()
		fmt.Printf("\nExamples:\n")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport system ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport godns ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport doh ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport dot ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport tcp ...")
		fmt.Printf("%s\n", "  ./httpclient -dns-transport udp [-dns-udp-server <addr>:<port>] ...")
		fmt.Printf("\nWe'll select a suitable backend for each transport. Note\n")
		fmt.Printf("that this only works on Unix.\n")
		return nil
	}
	if *flagDNSTransport == "system" {
		err = client.ConfigureDNS("system", "")
	} else if *flagDNSTransport == "godns" {
		err = client.ConfigureDNS("godns", "")
	} else if *flagDNSTransport == "udp" {
		err = client.ConfigureDNS("udp", *flagDNSUDPServer)
	} else if *flagDNSTransport == "tcp" {
		err = client.ConfigureDNS("tcp", "8.8.8.8:53")
	} else if *flagDNSTransport == "dot" {
		err = client.ConfigureDNS("dot", "dns.quad9.net")
	} else if *flagDNSTransport == "doh" {
		err = client.ConfigureDNS("doh", "https://cloudflare-dns.com/dns-query")
	} else if *flagDNSTransport != "" {
		err = errors.New("invalid -dns-transport argument")
	}
	rtx.PanicOnError(err, "cannot configure DNS transport")
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
