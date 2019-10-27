package main

import (
	"testing"

	"github.com/ooni/netx/cmd/common"
)

func TestIntegration(t *testing.T) {
	main()
}

func TestIntegrationBatch(t *testing.T) {
	*flagBatch = true
	defer func() {
		*flagBatch = false
	}()
	main()
}
func TestHelp(t *testing.T) {
	*common.FlagHelp = true
	err := mainfunc()
	*common.FlagHelp = false
	if err != nil {
		t.Fatal(err)
	}
}

func TestSystemTransport(t *testing.T) {
	*flagDNSServer = "system:///"
	defer func() {
		*flagDNSServer = ""
	}()
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUDPTransport(t *testing.T) {
	*flagDNSServer = "udp://1.1.1.1:53"
	defer func() {
		*flagDNSServer = ""
	}()
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestTCPTransport(t *testing.T) {
	*flagDNSServer = "tcp://8.8.8.8:53"
	defer func() {
		*flagDNSServer = ""
	}()
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoTTransport(t *testing.T) {
	*flagDNSServer = "dot://dns.quad9.net"
	defer func() {
		*flagDNSServer = ""
	}()
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoHTransport(t *testing.T) {
	*flagDNSServer = "https://cloudflare-dns.com/dns-query"
	defer func() {
		*flagDNSServer = ""
	}()
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestInvalidTransport(t *testing.T) {
	*flagDNSServer = "invalid"
	defer func() {
		*flagDNSServer = ""
	}()
	err := mainfunc()
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestParseError(t *testing.T) {
	*flagDNSServer = "inva@lid://"
	defer func() {
		*flagDNSServer = ""
	}()
	err := mainfunc()
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestInvalidURL(t *testing.T) {
	*flagURL = "\t"
	defer func() {
		*flagURL = "" // restore default
	}()
	err := mainfunc()
	if err == nil {
		t.Fatal("expected an error here")
	}
}
