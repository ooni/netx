// tlsconnect performs a TLS connect.
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
		flagAddress      = flag.String("tlsconnect-address", "example.com:443", "Address to connect to")
		flagDNSAddress   = flag.String("tlsconnect-dns-address", "", "Transport dependent address")
		flagDNSTransport = flag.String("tlsconnect-dns-transport", "system", "DNS transport")
		flagSNI          = flag.String("tlsconnect-sni", "", "Force specific SNI")
		flagTimeout      = flag.Duration("tlsconnect-timeout", 60*time.Second, "Overall timeout")
	)
	flag.Parse()
	log.SetHandler(cli.Default)
	log.SetLevel(log.DebugLevel)
	ctx, cancel := context.WithTimeout(
		context.Background(), *flagTimeout,
	)
	defer cancel()
	results := porcelain.TLSConnect(ctx, porcelain.TLSConnectConfig{
		Address:          *flagAddress,
		DNSServerAddress: *flagDNSAddress,
		DNSServerNetwork: *flagDNSTransport,
		Handler:          logger.NewHandler(log.Log),
		SNI:              *flagSNI,
	})
	data, err := json.MarshalIndent(results, "", "  ")
	rtx.Must(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
