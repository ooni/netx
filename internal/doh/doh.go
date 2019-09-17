// Package doh implements DNS over HTTPS
package doh

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/bassosimone/netx/internal/dox"
)

// NewConn creates a new net.PacketConn compatible connection that
// will forward DNS queries to the specified DoH server.
func NewConn(url string) (net.Conn, error) {
	return net.Conn(dox.NewConn(func(b []byte) dox.Result {
		return do(http.DefaultClient, url, b)
	})), nil
}

func do(client *http.Client, url string, b []byte) (out dox.Result) {
	var req *http.Request
	req, out.Err = http.NewRequest("POST", url, bytes.NewReader(b))
	if out.Err != nil {
		return
	}
	req.Header.Set("content-type", "application/dns-message")
	var resp *http.Response
	resp, out.Err = client.Do(req)
	if out.Err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		out.Err = errors.New("doh: server returned error")
		return
	}
	if resp.Header.Get("content-type") != "application/dns-message" {
		out.Err = errors.New("doh: invalid content-type")
		return
	}
	out.Data, out.Err = ioutil.ReadAll(resp.Body)
	return
}
