package model

import (
	"context"
	"crypto/tls"
	"testing"
	"time"
)

func TestNewTLSConnectionState(t *testing.T) {
	conn, err := tls.Dial("tcp", "www.google.com:443", nil)
	if err != nil {
		t.Fatal(err)
	}
	state := NewTLSConnectionState(conn.ConnectionState())
	if len(state.PeerCertificates) < 1 {
		t.Fatal("too few certificates")
	}
	if state.Version < tls.VersionSSL30 || state.Version > 0x0304 /*tls.VersionTLS13*/ {
		t.Fatal("unexpected TLS version")
	}
}

func TestMeasurementRoot(t *testing.T) {
	ctx := context.Background()
	if ContextMeasurementRoot(ctx) != nil {
		t.Fatal("unexpected value for ContextMeasurementRoot")
	}
	if ContextMeasurementRootOrDefault(ctx) == nil {
		t.Fatal("unexpected value ContextMeasurementRootOrDefault")
	}
	handler := &dummyHandler{}
	root := &MeasurementRoot{
		Handler:   handler,
		Beginning: time.Time{},
	}
	ctx = WithMeasurementRoot(ctx, root)
	v := ContextMeasurementRoot(ctx)
	if v != root {
		t.Fatal("unexpected ContextMeasurementRoot value")
	}
	v = ContextMeasurementRootOrDefault(ctx)
	if v != root {
		t.Fatal("unexpected ContextMeasurementRoot value")
	}
}

func TestMeasurementRootWithMeasurementRootPanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	ctx := context.Background()
	ctx = WithMeasurementRoot(ctx, nil)
}
