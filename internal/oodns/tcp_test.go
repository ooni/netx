package oodns

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestIntegrationTCPSuccess(t *testing.T) {
	transport := NewTransportTCP(
		"dns.quad9.net:53", (&net.Dialer{}).DialContext,
	)
	if err := threeRounds(transport); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationTCPFailure(t *testing.T) {
	transport := NewTransportTCP(
		"dns.quad9.net:53",
		func(ctx context.Context, network, address string) (net.Conn, error) {
			return nil, errors.New("mocked error")
		},
	)
	if err := threeRounds(transport); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationTCPSetDeadlineError(t *testing.T) {
	transport := NewTransportTCP(
		"dns.quad9.net:53",
		func(ctx context.Context, network, address string) (net.Conn, error) {
			return fakeconn{
				setDeadlineError: errors.New("mocked error"),
			}, nil
		},
	)
	if err := threeRounds(transport); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationTCPWriteError(t *testing.T) {
	transport := NewTransportTCP(
		"dns.quad9.net:53",
		func(ctx context.Context, network, address string) (net.Conn, error) {
			return fakeconn{
				writeError: errors.New("mocked error"),
			}, nil
		},
	)
	if err := threeRounds(transport); err == nil {
		t.Fatal("expected an error here")
	}
}

type firstfailingconn struct {
	fakeconn
}

func (ffc firstfailingconn) Read(b []byte) (n int, err error) {
	return 0, errors.New("mocked error")
}

func TestIntegrationTCPFirstReadError(t *testing.T) {
	transport := NewTransportTCP(
		"dns.quad9.net:53",
		func(ctx context.Context, network, address string) (net.Conn, error) {
			return firstfailingconn{}, nil
		},
	)
	if err := threeRounds(transport); err == nil {
		t.Fatal("expected an error here")
	}
}

type secondfailingconn struct {
	fakeconn
}

func (sfc secondfailingconn) Read(b []byte) (n int, err error) {
	if len(b) == 2 {
		b[0], b[1] = 0, 0
		return 2, nil
	}
	return 0, errors.New("mocked error")
}

func TestIntegrationTCPZeroLengthCase(t *testing.T) {
	transport := NewTransportTCP(
		"dns.quad9.net:53",
		func(ctx context.Context, network, address string) (net.Conn, error) {
			return secondfailingconn{}, nil
		},
	)
	if err := threeRounds(transport); err == nil {
		t.Fatal("expected an error here")
	}
}

type thirdfailingconn struct {
	fakeconn
}

func (sfc thirdfailingconn) Read(b []byte) (n int, err error) {
	if len(b) == 2 {
		b[0], b[1] = 1, 0
		return 2, nil
	}
	return 0, errors.New("mocked error")
}

func TestIntegrationTCPSecondReadError(t *testing.T) {
	transport := NewTransportTCP(
		"dns.quad9.net:53",
		func(ctx context.Context, network, address string) (net.Conn, error) {
			return thirdfailingconn{}, nil
		},
	)
	if err := threeRounds(transport); err == nil {
		t.Fatal("expected an error here")
	}
}
