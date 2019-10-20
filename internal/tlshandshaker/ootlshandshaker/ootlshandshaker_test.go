package ootlshandshaker

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
)

func TestIntegrationSuccess(t *testing.T) {
	handshaker := New()
	conn, err := (&net.Dialer{}).Dial("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	tlsconn, err := handshaker.Do(
		context.Background(),
		conn,
		&tls.Config{},
		"youtube.com", // 🙃
	)
	if err != nil {
		t.Fatal(err)
	}
	if tlsconn == nil {
		t.Fatal("expected non-nil tslconn")
	}
	tlsconn.Close()
}

func TestIntegrationTLSHandshakeFailure(t *testing.T) {
	handshaker := New()
	conn, err := (&net.Dialer{}).Dial("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	tlsconn, err := handshaker.Do(
		context.Background(),
		conn,
		&tls.Config{},
		"x.org",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if tlsconn == nil {
		t.Fatal("expected non-nil tslconn")
	}
	tlsconn.Close()
}

func TestIntegrationContextDeadline(t *testing.T) {
	handshaker := New()
	conn, err := (&net.Dialer{}).Dial("tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail now
	tlsconn, err := handshaker.Do(
		ctx, conn, &tls.Config{}, "x.org",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if tlsconn != nil {
		t.Fatal("expected nil tslconn")
	}
}
