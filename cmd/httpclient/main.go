package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/bassosimone/netx/httpx"
)

// XXX: better handling of HTTP bodies and request IDs
// XXX: better handling of logging

func main() {
	client := httpx.NewClient()
	client.Dialer.EnableTiming = true
	for _, URL := range os.Args[1:] {
		client.Get(URL)
	}
	data, err := json.Marshal(client.HTTPEvents())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", string(data))
	data, err = json.Marshal(client.NetEvents())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", string(data))
}
