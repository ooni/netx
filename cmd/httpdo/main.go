// httpdo performs a HTTP request
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
		flagDNSAddress   = flag.String("httpdo-dns-address", "", "Transport dependent address")
		flagDNSTransport = flag.String("httpdo-dns-transport", "system", "DNS transport")
		flagMethod       = flag.String("httpdo-method", "GET", "Method to use")
		flagNoVerify     = flag.Bool("httpdo-no-verify", false, "Skip TLS verification")
		flagTimeout      = flag.Duration("httpdo-timeout", 60*time.Second, "Overall timeout")
		flagURL          = flag.String("httpdo-url", "https://ooni.io", "URL to use")
	)
	flag.Parse()
	log.SetHandler(cli.Default)
	log.SetLevel(log.DebugLevel)
	ctx, cancel := context.WithTimeout(
		context.Background(), *flagTimeout,
	)
	defer cancel()
	results := porcelain.HTTPDo(ctx, porcelain.HTTPDoConfig{
		DNSServerAddress:   *flagDNSAddress,
		DNSServerNetwork:   *flagDNSTransport,
		Method:             *flagMethod,
		Handler:            logger.NewHandler(log.Log),
		InsecureSkipVerify: *flagNoVerify,
		URL:                *flagURL,
	})
	data, err := json.MarshalIndent(results, "", "  ")
	rtx.Must(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
