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

// Client is a DoH client
type Client struct {
	client *http.Client
	dialer *dialerapi.Dialer
	url    *url.URL
}

// NewClient creates a new client.
func NewClient(dialer *dialerapi.Dialer, address string) (*Client, error) {
	URL, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	child := dialerapi.NewDialer(dialer.Beginning, dialer.Handler)
	transport := httptransport.NewTransport(dialer.Beginning, dialer.Handler)
	// this duplicates some logic from httpx/httpx.go
	child.TLSConfig = transport.TLSClientConfig
	transport.Dial = child.Dial
	transport.DialContext = child.DialContext
	transport.DialTLS = child.DialTLS
	transport.MaxConnsPerHost = 1 // seems to be better for cloudflare
	client := &http.Client{Transport: transport}
	return &Client{
		dialer: dialer,
		client: client,
		url:    URL,
	}, nil
}

// NewResolver creates a new resolver that uses the specified server
// URL, and SNI, to resolve domain names using DoH.
func (clnt *Client) NewResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(c context.Context, n string, a string) (net.Conn, error) {
			return clnt.NewConn()
		},
	}
}

// NewConn creates a new doh pseudo-conn.
func (clnt *Client) NewConn() (net.Conn, error) {
	return dox.NewConn(clnt.dialer.Beginning, clnt.dialer.Handler, func(b []byte) dox.Result {
		return clnt.do(b)
	}), nil
}

// RoundTrip implements the dnsx.RoundTripper interface
func (clnt *Client) RoundTrip(query []byte) (reply []byte, err error) {
	out := clnt.do(query)
	reply = out.Data
	err = out.Err
	return
}

func (clnt *Client) do(b []byte) (out dox.Result) {
	req := &http.Request{
		Method:        "POST",
		URL:           clnt.url,
		Header:        http.Header{},
		Body:          ioutil.NopCloser(bytes.NewReader(b)),
		ContentLength: int64(len(b)),
	}
	req.Header.Set("content-type", "application/dns-message")
	var resp *http.Response
	resp, out.Err = clnt.client.Do(req)
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
