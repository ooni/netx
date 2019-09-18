// Package doh implements DNS over HTTPS
package doh

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/bassosimone/netx/internal/dialerapi"
	"github.com/bassosimone/netx/internal/dox"
	"github.com/bassosimone/netx/internal/httptransport"
)

// NewResolver creates a new resolver that uses the specified server
// URL, and SNI, to resolve domain names using DoH.
func NewResolver(dialer *dialerapi.Dialer, URL *url.URL) *net.Resolver {
	child := dialerapi.NewDialer(dialer.Beginning, dialer.C)
	transport := httptransport.NewTransport(dialer.Beginning, dialer.C)
	// this duplicates some logic from httpx/httpx.go
	child.TLSConfig = transport.TLSClientConfig
	transport.Dial = child.Dial
	transport.DialContext = child.DialContext
	transport.DialTLS = child.DialTLS
	transport.MaxConnsPerHost = 1 // seems to be better for cloudflare
	client := &http.Client{Transport: transport}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(c context.Context, n string, a string) (net.Conn, error) {
			return newConn(dialer, client, URL)
		},
	}
}

func newConn(dialer *dialerapi.Dialer, client *http.Client, URL *url.URL) (net.Conn, error) {
	return dox.NewConn(dialer.Beginning, dialer.C, func(b []byte) dox.Result {
		return do(client, URL, b)
	}), nil
}

func do(client *http.Client, URL *url.URL, b []byte) (out dox.Result) {
	req := &http.Request{
		Method:        "POST",
		URL:           URL,
		Header:        http.Header{},
		Body:          ioutil.NopCloser(bytes.NewReader(b)),
		ContentLength: int64(len(b)),
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
