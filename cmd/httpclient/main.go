package main

import (
	//"encoding/json"
	//"fmt"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/bassosimone/netx/httpx"
)

type baseLogger struct{
	begin time.Time
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
	fmt.Printf("[%10d] %s\n", time.Now().Sub(bl.begin)/time.Microsecond, msg)
}

// XXX: better handling of HTTP bodies and request IDs
// XXX: better handling of logging

func main() {
	client := httpx.NewClient()
	log := baseLogger{
		begin: time.Now(),
	}
	client.Dialer.Logger = log
	client.Dialer.EnableTiming = true
	client.Tracer.EventsContainer.Logger = log
	for _, URL := range os.Args[1:] {
		resp, err := client.Get(URL)
		if err != nil {
			continue
		}
		ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
	/*
		data, err := json.Marshal(client.HTTPEvents())
		if err != nil {
			log.WithError(err).Fatal("json.Marshal failed")
		}
		fmt.Printf("%s\n", string(data))
		data, err = json.Marshal(client.NetEvents())
		if err != nil {
			log.WithError(err).Fatal("json.Marshal failed")
		}
		fmt.Printf("%s\n", string(data))
	*/
}
