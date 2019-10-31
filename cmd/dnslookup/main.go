// dnslookup performs a DNS lookup.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/m-lab/go/rtx"
	"github.com/ooni/netx/x/logger"
	"github.com/ooni/netx/x/porcelain"
)

func main() {
	var (
		flagAddress   = flag.String("dnslookup-address", "", "Transport dependent address")
		flagName      = flag.String("dnslookup-name", "ooni.io", "Name to lookup")
		flagTransport = flag.String("dnslookup-transport", "system", "DNS transport")
	)
	flag.Parse()
	log.SetHandler(cli.Default)
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()
	results, err := porcelain.DNSLookup(ctx, porcelain.DNSLookupConfig{
		Handler:       logger.NewHandler(log.Log),
		Hostname:      *flagName,
		ServerAddress: *flagAddress,
		ServerNetwork: *flagTransport,
	})
	rtx.Must(err, "porcelain.DNSLookup failed")
	data, err := json.MarshalIndent(results, "", "  ")
	rtx.Must(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
