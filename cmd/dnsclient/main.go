// dnsclient is a simple DNS command line client. The first argument
// must be the type of resolver you want to create ("udp", "tcp", "doh",
// or "dot") and the remaining arguments are names to resolve.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/bassosimone/netx"
	"github.com/bassosimone/netx/internal/testingx"
	"github.com/bassosimone/netx/model"
	"github.com/miekg/dns"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: dnsclient udp|tcp|dot|doh names...\n")
		os.Exit(1)
	}
	ch := make(chan model.Measurement)
	cancel := testingx.SpawnLogger(ch)
	defer cancel()
	dialer := netx.NewDialer(ch)
	address, err := suitableAddress(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	conn, err := dialer.DialDoX(os.Args[1], address)
	if err != nil {
		log.Fatal(err)
	}
	for _, address := range os.Args[2:] {
		m1 := new(dns.Msg)
		m1.Id = dns.Id()
		m1.RecursionDesired = true
		m1.Question = make([]dns.Question, 1)
		m1.Question[0] = dns.Question{
			Name:   dns.Fqdn(address),
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		}
		m1.SetEdns0(4096, true)
		data, err := m1.Pack()
		if err != nil {
			log.Printf("%s\n", err.Error())
			continue
		}
		_, err = conn.Write(data)
		if err != nil {
			log.Printf("%s\n", err.Error())
			continue
		}
		data = make([]byte, 1<<16)
		n, err := conn.Read(data)
		if err != nil {
			log.Printf("%s\n", err.Error())
			continue
		}
		data = data[:n]
		err = m1.Unpack(data)
		if err != nil {
			log.Printf("%s\n", err.Error())
			continue
		}
		fmt.Printf("\n%s\n", m1.String())
	}
}

func suitableAddress(network string) (string, error) {
	if network == "udp" {
		return "1.1.1.1:53", nil
	}
	if network == "tcp" {
		return "8.8.8.8:53", nil
	}
	if network == "dot" {
		return "dns.quad9.net", nil
	}
	if network == "doh" {
		return "https://cloudflare-dns.com/dns-query", nil
	}
	return "", errors.New("unknown network")
}
