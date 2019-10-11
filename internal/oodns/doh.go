package oodns

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/ooni/netx/dnsx"
)

type dohTransport struct {
	clientDo func(req *http.Request) (*http.Response, error)
	url      string
}

// NewTransportDoH creates a new DoH transport.
func NewTransportDoH(client *http.Client, URL string) dnsx.RoundTripper {
	return &dohTransport{
		clientDo: client.Do,
		url:      URL,
	}
}

func (t *dohTransport) RoundTrip(query []byte) ([]byte, error) {
	return t.RoundTripContext(context.Background(), query)
}

func (t *dohTransport) RoundTripContext(
	ctx context.Context, query []byte,
) (reply []byte, err error) {
	req, err := http.NewRequest("POST", t.url, bytes.NewReader(query))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/dns-message")
	var resp *http.Response
	resp, err = t.clientDo(req.WithContext(ctx))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		// TODO(bassosimone): we should map the status code to a
		// proper Error in the DNS context.
		err = errors.New("doh: server returned error")
		return
	}
	if resp.Header.Get("content-type") != "application/dns-message" {
		err = errors.New("doh: invalid content-type")
		return
	}
	reply, err = ioutil.ReadAll(resp.Body)
	return
}
