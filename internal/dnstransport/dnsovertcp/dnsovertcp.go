// Package dnsovertcp implements DNS over TCP. It is possible to
// use both plaintext TCP and TLS.
package dnsovertcp

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/m-lab/go/rtx"
	"github.com/ooni/netx/internal/dialerapi"
	"github.com/ooni/netx/model"
)

// Transport is a DNS over TCP/TLS dnsx.RoundTripper.
//
// As a known bug, this implementation always creates a new connection
// for each incoming query, thus increasing the response delay.
type Transport struct {
	// Dialer is the dialer to use.
	Dialer *dialerapi.Dialer

	// Hostname is the hostname of the service.
	Hostname string

	// LookupHost allows you to override the code used to lookup
	// the address of the DoT server domain name.
	LookupHost func(host string) (addrs []string, err error)

	// NoTLS indicates that we don't want to use TLS.
	NoTLS bool

	// Port is the port of the service.
	Port string

	// address is the resolved address of the service.
	address string

	// init indicates whether we've initialized
	init bool

	// mutex makes initialize idempotent.
	mutex sync.Mutex
}

// NewTransport creates a new Transport
func NewTransport(beginning time.Time, handler model.Handler, hostname string) *Transport {
	dialer := dialerapi.NewDialer(beginning, handler)
	return &Transport{
		Dialer:   dialer,
		Hostname: hostname,
	}
}

func (t *Transport) initUnlocked() (err error) {
	if t.LookupHost == nil {
		t.LookupHost = net.LookupHost
	}
	if t.address == "" {
		if net.ParseIP(t.Hostname) == nil {
			var addrs []string
			addrs, err = t.LookupHost(t.Hostname)
			if err != nil {
				return err
			}
			if len(addrs) < 1 {
				return errors.New("dnsovertcp: net.LookupHost: empty reply")
			}
			t.address = addrs[0]
		} else {
			t.address = t.Hostname
		}
	}
	if t.NoTLS == false {
		if t.Port == "" {
			t.Port = "853"
		}
		if t.Dialer.TLSConfig != nil {
			t.Dialer.TLSConfig = t.Dialer.TLSConfig.Clone()
		} else {
			t.Dialer.TLSConfig = &tls.Config{}
		}
		t.Dialer.TLSConfig.ServerName = t.Hostname
	} else if t.Port == "" {
		t.Port = "53"
	}
	return nil
}

func (t *Transport) initialize() (err error) {
	t.mutex.Lock()
	if !t.init {
		t.init = true
		err = t.initUnlocked()
	}
	t.mutex.Unlock()
	return
}

// RoundTrip sends a request and receives a response.
func (t *Transport) RoundTrip(query []byte) ([]byte, error) {
	var (
		conn net.Conn
		err  error
	)
	err = t.initialize()
	if err != nil {
		return nil, err
	}
	if t.NoTLS == false {
		conn, err = t.Dialer.DialTLS(
			"tcp", net.JoinHostPort(t.address, t.Port),
		)
	} else {
		conn, err = t.Dialer.Dial(
			"tcp", net.JoinHostPort(t.address, t.Port),
		)
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return t.RoundTripWithConn(conn, query)
}

// RoundTripWithConn performs the DNS round trip with a connection.
func (t *Transport) RoundTripWithConn(conn net.Conn, query []byte) (reply []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			reply = nil // we already got the error just clear the reply
		}
	}()
	err = conn.SetDeadline(time.Now().Add(10 * time.Second))
	rtx.PanicOnError(err, "conn.SetDeadline failed")
	// Write request
	writer := bufio.NewWriter(conn)
	err = writer.WriteByte(byte(len(query) >> 8))
	rtx.PanicOnError(err, "writer.WriteByte failed for first byte")
	err = writer.WriteByte(byte(len(query)))
	rtx.PanicOnError(err, "writer.WriteByte failed for second byte")
	_, err = writer.Write(query)
	rtx.PanicOnError(err, "writer.Write failed for query")
	err = writer.Flush()
	rtx.PanicOnError(err, "writer.Flush failed")
	// Read response
	header := make([]byte, 2)
	_, err = io.ReadFull(conn, header)
	rtx.PanicOnError(err, "io.ReadFull failed")
	length := int(header[0])<<8 | int(header[1])
	reply = make([]byte, length)
	_, err = io.ReadFull(conn, reply)
	rtx.PanicOnError(err, "io.ReadFull failed")
	return reply, nil
}
