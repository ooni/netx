// tlsclient is a simple TLS command line client.
//
// Usage:
//
//   tlsclient -address address -sni sni
//
//   tlsclient -help
//
// Examples:
//
//   ./tlsclient -address example.com:443 -sni ooni.io
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/m-lab/go/rtx"
	"github.com/ooni/netx/cmd/common"
	"github.com/ooni/netx/x/logger"
	"github.com/ooni/netx/x/porcelain"
)

var (
	flagAddress = flag.String("address", "example.com:443", "Address to connect to")
)

func main() {
	flag.Parse()
	if *common.FlagHelp {
		flag.CommandLine.SetOutput(os.Stdout)
		fmt.Printf("Usage: tlsclient [flags]\n")
		flag.PrintDefaults()
		fmt.Printf("\nExamples:\n")
		fmt.Printf("%s\n", "  ./tlsclient -address example.com:443 -sni ooni.io")
		return
	}
	log.SetHandler(cli.Default)
	log.SetLevel(log.DebugLevel)
	measurements, _ := porcelain.TLSConnect(
		logger.NewHandler(log.Log), *flagAddress, *common.FlagSNI,
	)
	prettyprint(measurements)
}

func prettyprint(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	rtx.Must(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
