package main

import (
	"testing"

	"github.com/ooni/netx/cmd/common"
)

func TestIntegration(t *testing.T) {
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
	*flagDNSTransport = "system"
	defer func() {
		*flagDNSTransport = ""
	}()
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoDNSTransport(t *testing.T) {
	*flagDNSTransport = "godns"
	defer func() {
		*flagDNSTransport = ""
	}()
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUDPTransport(t *testing.T) {
	*flagDNSTransport = "udp"
	defer func() {
		*flagDNSTransport = ""
	}()
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestTCPTransport(t *testing.T) {
	*flagDNSTransport = "tcp"
	defer func() {
		*flagDNSTransport = ""
	}()
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoTTransport(t *testing.T) {
	*flagDNSTransport = "dot"
	defer func() {
		*flagDNSTransport = ""
	}()
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoHTransport(t *testing.T) {
	*flagDNSTransport = "doh"
	defer func() {
		*flagDNSTransport = ""
	}()
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestInvalidTransport(t *testing.T) {
	*flagDNSTransport = "invalid"
	defer func() {
		*flagDNSTransport = ""
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
