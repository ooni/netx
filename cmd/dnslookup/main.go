// dnslookup performs a DNS lookup.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

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
		flagTimeout   = flag.Duration("dnslookup-timeout", 60*time.Second, "Overall timeout")
		flagTransport = flag.String("dnslookup-transport", "system", "DNS transport")
	)
	flag.Parse()
	log.SetHandler(cli.Default)
	log.SetLevel(log.DebugLevel)
	ctx, cancel := context.WithTimeout(
		context.Background(), *flagTimeout,
	)
	defer cancel()
	results := porcelain.DNSLookup(ctx, porcelain.DNSLookupConfig{
		Handler:       logger.NewHandler(log.Log),
		Hostname:      *flagName,
		ServerAddress: *flagAddress,
		ServerNetwork: *flagTransport,
	})
	data, err := json.MarshalIndent(results, "", "  ")
	rtx.Must(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
