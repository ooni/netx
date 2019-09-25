// dnsclient is a simple DNS command line client.
//
// Usage:
//
//   dnsclient -type Addr|CNAME|Host|MX|NS -name <name>
//             -transport tcp|udp|dot|doh
//             -endpoint <transport-specific-endpoint>
//
// The default is to use the udp transport. For each transport
// we use a specific default resolver.
//
// We emit JSONL messages on the stdout showing what we are
// currently doing. We also print the final result on the stdout.
//
// Examples:
//
//   ./dnsclient -transport doh -endpoint https://cloudflare-dns.com/dns-query ...
//   ./dnsclient -transport dot -endpoint dns.quad9.net ...
//   ./dnsclient -transport tcp -endpoint 8.8.8.8:53 ...
//   ./dnsclient -transport udp -endpoint 1.1.1.1:53 ...
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ooni/netx"
	"github.com/ooni/netx/dnsx"
	"github.com/ooni/netx/handlers"
)

func main() {
	dialer := netx.NewDialer(handlers.StdoutHandler)
	var (
		flagName      = flag.String("name", "ooni.io", "Name to query for")
		flagEndpoint  = flag.String("endpoint", "8.8.8.8:53", "Transport endpoint")
		flagHelp      = flag.Bool("h", false, "Print usage")
		flagTransport = flag.String("transport", "udp", "Transport to use")
		flagType      = flag.String("type", "Host", "Query type")
		err           error
		resolver      dnsx.Client
	)
	flag.Parse()
	if *flagHelp {
		flag.CommandLine.SetOutput(os.Stdout)
		fmt.Printf("Usage: dnsclient [flags]\n")
		flag.PrintDefaults()
		fmt.Printf("\nExamples:\n")
		fmt.Printf("%s\n", "  ./dnsclient -transport doh -endpoint https://cloudflare-dns.com/dns-query ...")
		fmt.Printf("%s\n", "  ./dnsclient -transport dot -endpoint dns.quad9.net ...")
		fmt.Printf("%s\n", "  ./dnsclient -transport tcp -endpoint 8.8.8.8:53 ...")
		fmt.Printf("%s\n", "  ./dnsclient -transport udp -endpoint 1.1.1.1:53 ...")
		os.Exit(0)
	}
	resolver, err = dialer.NewResolver(*flagTransport, *flagEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	if *flagType == "Addr" {
		names, err := resolver.LookupAddr(ctx, *flagName)
		prettyprint(names)
		prettyprint(err)
	} else if *flagType == "CNAME" {
		cname, err := resolver.LookupCNAME(ctx, *flagName)
		prettyprint(cname)
		prettyprint(err)
	} else if *flagType == "Host" {
		addrs, err := resolver.LookupHost(ctx, *flagName)
		prettyprint(addrs)
		prettyprint(err)
	} else if *flagType == "MX" {
		recs, err := resolver.LookupMX(ctx, *flagName)
		prettyprint(recs)
		prettyprint(err)
	} else if *flagType == "NS" {
		recs, err := resolver.LookupNS(ctx, *flagName)
		prettyprint(recs)
		prettyprint(err)
	} else {
		log.Fatal("unsupported query type")
	}
}

func prettyprint(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", string(data))
}
