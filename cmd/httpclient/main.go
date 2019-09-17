// httpclient is a simple HTTP command line client. It will fetch all the
// URLs passed to the command line. While doing that, it will log http and
// network level events on the standard error. When done, it will print
// on the standard output the observed events as JSON.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bassosimone/netx/httpx"
)

type baseLogger struct {
	beginning time.Time
}

func (bl baseLogger) Debug(msg string) {
	bl.log(msg)
}
func (bl baseLogger) Debugf(format string, v ...interface{}) {
	bl.logf(format, v...)
}
func (bl baseLogger) Info(msg string) {
	bl.log(msg)
}
func (bl baseLogger) Infof(format string, v ...interface{}) {
	bl.logf(format, v...)
}
func (bl baseLogger) Warn(msg string) {
	bl.log(msg)
}
func (bl baseLogger) Warnf(format string, v ...interface{}) {
	bl.logf(format, v...)
}
func (bl baseLogger) logf(format string, v ...interface{}) {
	bl.log(fmt.Sprintf(format, v...))
}
func (bl baseLogger) log(msg string) {
	fmt.Fprintf(os.Stderr, "[%10d] %s\n",
		time.Now().Sub(bl.beginning)/time.Microsecond, msg)
}

func mustDump(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", string(data))
}

func fetch(client *http.Client, url string) {
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	ioutil.ReadAll(resp.Body)
	resp.Body.Close()
}

func main() {
	client, err := httpx.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	client.SetLogger(baseLogger{
		beginning: client.Beginning(),
	})
	client.EnableFullTiming()
	//client.Dialer().ConfigureDNS("udp", "1.1.1.1:53")
	//client.Dialer().ConfigureDNS("tcp", "8.8.8.8:53")
	//client.Dialer().ConfigureDNS("dot", "dns.quad9.net")
	//client.Dialer().ConfigureDNS("doh", "https://cloudflare-dns.com/dns-query")
	for _, url := range os.Args[1:] {
		fetch(client.HTTPClient, url)
	}
	mustDump(client.PopNetMeasurements())
	mustDump(client.PopHTTPMeasurements())
}
