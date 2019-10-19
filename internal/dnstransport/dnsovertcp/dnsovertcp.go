// Package dnsovertcp implements DNS over TCP. It is possible to
// use both plaintext TCP and TLS.
package dnsovertcp

import (
	"bufio"
	"io"
	"net"
	"sync"
	"time"

	"github.com/m-lab/go/rtx"
)

// Transport is a DNS over TCP/TLS dnsx.RoundTripper.
type Transport struct {
	dial    func(network, address string) (net.Conn, error)
	address string
	mtx     sync.Mutex
}

// NewTransport creates a new Transport
func NewTransport(
	dial func(network, address string) (net.Conn, error),
	address string,
) *Transport {
	return &Transport{
		dial:    dial,
		address: address,
	}
}

type connInfo struct {
	conn   net.Conn
	latest time.Time
}

// Cache implementation - rationale: establishing a new TLS conn for every
// new query is slow. Let's reuse existing connections. Since we don't have
// a mechanism to close idle connections, instead roll out a private cache
// that we can remove any moment in favour of better mechanisms. The idea is
// to keep bounded the number of sockets used by DoT and prune periodically
// the cache so to close very old, stale connections.

type cacheInfo struct {
	cache  map[*Transport]*connInfo
	mtx    sync.Mutex
	latest time.Time
}

func newCacheInfo() *cacheInfo {
	return &cacheInfo{
		cache: make(map[*Transport]*connInfo),
	}
}

var cache = newCacheInfo()

func (c *cacheInfo) getconn(t *Transport) (conn net.Conn) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	info, ok := c.cache[t]
	var todelete []*Transport
	if ok {
		todelete = append(todelete, t)
		conn = info.conn
	}
	now := time.Now()
	if now.Sub(c.latest) > 10*time.Second {
		c.latest = now
		for other, info := range c.cache {
			if now.Sub(info.latest) > 10*time.Second {
				todelete = append(todelete, other)
				info.conn.Close()
			}
		}
	}
	for _, ref := range todelete {
		delete(c.cache, ref)
	}
	return
}

func (c *cacheInfo) putconn(t *Transport, conn net.Conn) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	info, ok := c.cache[t]
	if ok {
		info.conn.Close()
		delete(c.cache, t)
	}
	c.cache[t] = &connInfo{
		conn:   conn,
		latest: time.Now(),
	}
}

// RoundTrip sends a request and receives a response.
func (t *Transport) RoundTrip(query []byte) (reply []byte, err error) {
	// Implementation note: we serialize round trips because this
	// allows to simplify reusing connections.
	t.mtx.Lock()
	defer t.mtx.Unlock()
	conn := cache.getconn(t)
	if conn == nil {
		conn, err = t.dial("tcp", t.address)
		if err != nil {
			return nil, err
		}
	}
	reply, err = roundTripLocked(conn, query)
	if err == nil {
		cache.putconn(t, conn)
	} else {
		conn.Close()
	}
	return
}

func roundTripLocked(conn net.Conn, query []byte) (reply []byte, err error) {
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
