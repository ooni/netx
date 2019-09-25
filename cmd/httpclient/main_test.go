package main

import "testing"

func TestIntegration(t *testing.T) {
	main()
}

func TestHelp(t *testing.T) {
	*flagHelp = true
	err := mainfunc()
	*flagHelp = false
	if err != nil {
		t.Fatal(err)
	}
}

func TestUDPTransport(t *testing.T) {
	*flagDNSTransport = "udp"
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestTCPTransport(t *testing.T) {
	*flagDNSTransport = "tcp"
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoTTransport(t *testing.T) {
	*flagDNSTransport = "dot"
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoHTransport(t *testing.T) {
	*flagDNSTransport = "doh"
	err := mainfunc()
	if err != nil {
		t.Fatal(err)
	}
}

func TestInvalidTransport(t *testing.T) {
	*flagDNSTransport = "invalid"
	err := mainfunc()
	if err == nil {
		t.Fatal("expected an error here")
	}
}
