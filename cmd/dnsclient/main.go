// dnsclient is a simple DNS command line client.
//
// Usage:
//
//   dnsclient -type Addr|CNAME|Host|MX|NS -name <name>
//             -transport system|godns|tcp|udp|dot|doh
//             -endpoint <transport-specific-endpoint>
//
//   dnsclient -help
//
// The default is to use the system transport. For each transport
// we use a specific default resolver.
//
// We emit JSONL messages on the stdout showing what we are
// currently doing. We also print the final result on the stdout.
//
//
// Examples:
//
//   ./dnsclient -transport system ...
//   ./dnsclient -transport godns ...
//   ./dnsclient -transport doh -endpoint https://cloudflare-dns.com/dns-query ...
//   ./dnsclient -transport dot -endpoint dns.quad9.net ...
//   ./dnsclient -transport dot -endpoint 1.1.1.1:853 ...
//   ./dnsclient -transport tcp -endpoint 8.8.8.8:53 ...
//   ./dnsclient -transport udp -endpoint 1.1.1.1:53 ...
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/m-lab/go/rtx"
	"github.com/ooni/netx"
	"github.com/ooni/netx/cmd/common"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/model"
)

var (
	flagName      = flag.String("name", "ooni.io", "Name to query for")
	flagEndpoint  = flag.String("endpoint", "", "Transport endpoint")
	flagTransport = flag.String("transport", "system", "Transport to use")
	flagType      = flag.String("type", "Host", "Query type")
)

func mainWithContext(ctx context.Context) error {
	dialer := netx.NewDialer(handlers.StdoutHandler)
	var (
		addrs    []string
		cname    string
		err      error
		mxrecs   []*net.MX
		names    []string
		nsrecs   []*net.NS
		resolver model.DNSResolver
	)
	if *common.FlagHelp {
		flag.CommandLine.SetOutput(os.Stdout)
		fmt.Printf("Usage: dnsclient [flags]\n")
		flag.PrintDefaults()
		fmt.Printf("\nExamples:\n")
		fmt.Printf("%s\n", "  ./dnsclient -transport system ...")
		fmt.Printf("%s\n", "  ./dnsclient -transport godns ...")
		fmt.Printf("%s\n", "  ./dnsclient -transport doh -endpoint https://cloudflare-dns.com/dns-query ...")
		fmt.Printf("%s\n", "  ./dnsclient -transport dot -endpoint dns.quad9.net ...")
		fmt.Printf("%s\n", "  ./dnsclient -transport dot -endpoint 1.1.1.1:853 ...")
		fmt.Printf("%s\n", "  ./dnsclient -transport tcp -endpoint 8.8.8.8:53 ...")
		fmt.Printf("%s\n", "  ./dnsclient -transport udp -endpoint 1.1.1.1:53 ...")
		return nil
	}
	resolver, err = dialer.NewResolver(*flagTransport, *flagEndpoint)
	rtx.Must(err, "cannot create new resolver")
	if *flagType == "Addr" {
		names, err = resolver.LookupAddr(ctx, *flagName)
		prettyprint(names)
	} else if *flagType == "CNAME" {
		cname, err = resolver.LookupCNAME(ctx, *flagName)
		prettyprint(cname)
	} else if *flagType == "Host" {
		addrs, err = resolver.LookupHost(ctx, *flagName)
		prettyprint(addrs)
	} else if *flagType == "MX" {
		mxrecs, err = resolver.LookupMX(ctx, *flagName)
		prettyprint(mxrecs)
	} else if *flagType == "NS" {
		nsrecs, err = resolver.LookupNS(ctx, *flagName)
		prettyprint(nsrecs)
	} else {
		err = errors.New("unsupported query type")
	}
	prettyprint(err)
	return err
}

func main() {
	flag.Parse()
	err := mainWithContext(context.Background())
	rtx.Must(err, "mainWithContext failed")
}

func prettyprint(v interface{}) {
	data, err := json.Marshal(v)
	rtx.Must(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
