// httpdo performs a HTTP request
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
		flagDNSAddress   = flag.String("httpdo-dns-address", "", "Transport dependent address")
		flagDNSTransport = flag.String("httpdo-dns-transport", "system", "DNS transport")
		flagMethod       = flag.String("httpdo-method", "GET", "Method to use")
		flagURL          = flag.String("httpdo-url", "https://ooni.io", "URL to use")
	)
	flag.Parse()
	log.SetHandler(cli.Default)
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()
	results, err := porcelain.HTTPDo(ctx, porcelain.HTTPDoConfig{
		DNSServerAddress: *flagDNSAddress,
		DNSServerNetwork: *flagDNSTransport,
		Method:           *flagMethod,
		Handler:          logger.NewHandler(log.Log),
		URL:              *flagURL,
	})
	rtx.Must(err, "porcelain.HTTPDo failed")
	data, err := json.MarshalIndent(results, "", "  ")
	rtx.Must(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
