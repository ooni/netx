// httpclient is a simple HTTP command line client.
//
// Usage:
//
//   httpclient -batch -dns-server <URL> -sni <string> -url <URL>
//
//   httpclient -help
//
// The default is to use the system DNS. The `-dns-server <URL>` flag
// allows to choose what DNS transport to use (see below).
//
// With -batch we emit JSONL messages on the stdout showing what we are
// currently doing. Otherwise we emit user friendly log messages.
//
// Examples with `-dns-server <URL>`
//
//   ./httpclient -dns-server system:///
//   ./httpclient -dns-server udp://1.1.1.1:53
//   ./httpclient -dns-server tcp://8.8.8.8:53
//   ./httpclient -dns-server dot://dns.quad9.net
//   ./httpclient -dns-server dot://1.1.1.1:853
//   ./httpclient -dns-server https://cloudflare-dns.com/dns-query
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/m-lab/go/rtx"
	"github.com/ooni/netx/cmd/common"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/httpx"
	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/logger"
	"github.com/ooni/netx/x/nervousresolver"
	"github.com/ooni/netx/x/porcelain"
)

var (
	flagBatch     = flag.Bool("batch", false, "Emit JSON events")
	flagDNSServer = flag.String("dns-server", "system:///", "Server to use")
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
	log.SetLevel(log.DebugLevel)
	log.SetHandler(cli.Default)
	client := porcelain.NewHTTPXClient()
	if *common.FlagHelp {
		flag.CommandLine.SetOutput(os.Stdout)
		fmt.Printf("Usage: httpclient -dns-server <URL> -sni <string> -url <url>\n")
		flag.PrintDefaults()
		fmt.Printf("\nExamples with `-dns-server <URL>`:\n")
		fmt.Printf("  ./httpclient -dns-server udp://1.1.1.1:53\n")
		fmt.Printf("  ./httpclient -dns-server tcp://8.8.8.8:53\n")
		fmt.Printf("  ./httpclient -dns-server dot://dns.quad9.net\n")
		fmt.Printf("  ./httpclient -dns-server dot://1.1.1.1:853\n")
		fmt.Printf("  ./httpclient -dns-server https://cloudflare-dns.com/dns-query\n")
		fmt.Printf("  ./httpclient -dns-server x-nervous:///\n")
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
	} else if urlDNSServer.Scheme == "x-nervous" {
		// This is a new, experimental resolver, so it's using a more
		// direct and simple API for configuring a resolver.
		client.SetResolver(nervousresolver.Default)
	} else if *flagDNSServer != "" {
		err = errors.New("invalid -dns-server argument")
	}
	rtx.PanicOnError(err, "cannot configure DNS server")
	err = client.ForceSpecificSNI(*common.FlagSNI)
	rtx.PanicOnError(err, "cannot force specific SNI")
	err = fetch(client, *flagURL)
	rtx.PanicOnError(err, "cannot fetch specific URL")
	return
}

func fetch(client *httpx.Client, url string) (err error) {
	defer func() {
		if recover() != nil {
			// JUST KNOW WE ARRIVED HERE
		}
	}()
	measurements, err := porcelain.Get(
		makehandler(), client, url, "ooniprobe-netx/0.1.0",
	)
	data, _ := json.MarshalIndent(measurements, "", "  ")
	fmt.Printf("%s\n", data)
	rtx.PanicOnError(err, "porcelain.Get failed")
	return
}

func makehandler() model.Handler {
	if *flagBatch {
		return handlers.StdoutHandler
	}
	return logger.NewHandler(log.Log)
}
