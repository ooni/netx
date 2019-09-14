package netx

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"
)

func newFailingDialer() (dialer *MeasuringDialer) {
	dialer = NewMeasuringDialer(time.Now())
	dialer.EnableTiming = true
	dialer.LookupHost = func(ctx context.Context, host string) (addrs []string, err error) {
		return []string{"127.0.0.2", "127.0.0.3"}, nil
	}
	dialer.Dialer.Control = func(
		network string, address string, c syscall.RawConn,
	) error {
		return syscall.ECONNREFUSED
	}
	return
}

func checkFailingretryingDialerResult(
	conn net.Conn, err error, elapsed time.Duration,
	minElapsed time.Duration, maxElapsed time.Duration,
) error {
	if conn != nil {
		conn.Close()
		return errors.New("Expected to see nil connection")
	}
	if err == nil {
		return errors.New("Expected an error here")
	}
	if len(err.Error()) <= 0 {
		return errors.New("Expected an error with description")
	}
	if elapsed < minElapsed {
		return errors.New("Elapsed is too small")
	}
	if elapsed > maxElapsed {
		return errors.New("Elapsed is too large")
	}
	return nil
}

func TestConnectRetry(t *testing.T) {
	dialer := newFailingDialer()
	begin := time.Now()
	conn, err := dialer.Dial("tcp", "shelob.polito.it:80")
	err = checkFailingretryingDialerResult(
		conn, err, time.Now().Sub(begin), 11*time.Second,
		22*time.Second,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestConnectInterruptContext(t *testing.T) {
	dialer := newFailingDialer()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	begin := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", "shelob.polito.it:80")
	err = checkFailingretryingDialerResult(
		conn, err, time.Now().Sub(begin), 2*time.Second,
		5*time.Second,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestConnectSuccess(t *testing.T) {
	dialer := NewMeasuringDialer(time.Now())
	dialer.EnableTiming = true
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	conn, err := dialer.DialContext(ctx, "tcp", "www.google.com:80")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("Expected non-nil conn")
	}
	request := []byte("GET /humans.txt HTTP/1.0\r\n\r\n")
	n, err := conn.Write(request)
	if n != len(request) {
		t.Fatal("Unexpected number of bytes written")
	}
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 1<<20)
	n, err = conn.Read(buffer)
	if n <= 0 {
		t.Fatal("Unexpected number of bytes read")
	}
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}
