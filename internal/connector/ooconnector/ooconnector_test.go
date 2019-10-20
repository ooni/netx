package ooconnector

import (
	"context"
	"testing"
)

func TestIntegrationSuccess(t *testing.T) {
	conn, err := New().DialContext(context.Background(), "tcp", "8.8.8.8:53")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn here")
	}
	conn.Close()
}

func TestIntegrationFailureDomain(t *testing.T) {
	conn, err := New().DialContext(context.Background(), "tcp", "google.com:80")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestIntegrationFailureNoPort(t *testing.T) {
	conn, err := New().DialContext(context.Background(), "tcp", "8.8.8.8")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}
