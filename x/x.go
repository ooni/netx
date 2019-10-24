// Package x contains experimental code
package x

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/internal/dnstransport/dnsoverhttps"
	"github.com/ooni/netx/internal/dnstransport/dnsovertcp"
	"github.com/ooni/netx/internal/dnstransport/dnsoverudp"
	"github.com/ooni/netx/internal/oodns"
	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/dialer"
	"github.com/ooni/netx/x/httptransport"
	"github.com/ooni/netx/x/resolver"
	"github.com/ooni/netx/x/tlsdialer"
	"golang.org/x/net/http2"
)

// TLSConfigBuilder is a tls.Config builder
type TLSConfigBuilder struct {
	CABundlePath string
	SpecificSNI  string
}

// NewTLSConfigBuilder creates a new TLSConfigBuilder
func NewTLSConfigBuilder() *TLSConfigBuilder {
	return new(TLSConfigBuilder)
}

// Build builds a new *tls.Config
func (b *TLSConfigBuilder) Build() (*tls.Config, error) {
	config := new(tls.Config)
	if b.CABundlePath != "" {
		cert, err := ioutil.ReadFile(b.CABundlePath)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(cert)
		config.RootCAs = pool
	}
	if b.SpecificSNI != "" {
		config.ServerName = b.SpecificSNI
	}
	return config, nil
}

// ResolverBuilder is a resolver builder
type ResolverBuilder struct {
	Address   string
	Beginning time.Time
	Handler   model.Handler
	TLSConfig *tls.Config
	Type      string
}

// NewResolverBuilder creates a new resolver builder
func NewResolverBuilder() *ResolverBuilder {
	return &ResolverBuilder{
		Address:   "",
		Beginning: time.Now(),
		Handler:   handlers.StdoutHandler,
		TLSConfig: new(tls.Config),
		Type:      "system",
	}
}

// UseSystem configures using the system resolver
func (b *ResolverBuilder) UseSystem() {
	b.Address, b.Type = "", "system"
}

// UseUDP configures using a specific UDP server
func (b *ResolverBuilder) UseUDP(address string) {
	b.Address, b.Type = address, "udp"
}

// UseTCP configures using a specific TCP server
func (b *ResolverBuilder) UseTCP(address string) {
	b.Address, b.Type = address, "tcp"
}

// UseTLS configures using a specific DNS over TLS server
func (b *ResolverBuilder) UseTLS(address string) {
	b.Address, b.Type = address, "dot"
}

// UseHTTPS configures using a specific DNS over HTTPS server
func (b *ResolverBuilder) UseHTTPS(address string) {
	b.Address, b.Type = address, "doh"
}

// UseURL configures the resolver builder using a specific URL
func (b *ResolverBuilder) UseURL(URL string) error {
	u, err := url.Parse(URL)
	if err != nil {
		return err
	}
	if u.Scheme == "system" {
		b.UseSystem()
		return nil
	}
	if u.Scheme == "udp" || u.Scheme == "tcp" || u.Scheme == "dot" {
		b.UseUDP(u.Host)
		return nil
	}
	if u.Scheme == "https" {
		b.UseHTTPS(URL)
		return nil
	}
	return errors.New("unsupported URL scheme")
}

// Build builds a new resolver
func (b *ResolverBuilder) Build() (reso model.Resolver, err error) {
	if b.Type == "system" {
		reso = new(net.Resolver)
	} else if b.Type == "udp" {
		parentreso := resolver.New(b.Beginning, b.Handler, new(net.Resolver))
		dialer := NewDialer(b.Beginning, b.Handler, parentreso, true)
		reso = oodns.NewClient(b.Handler, dnsoverudp.NewTransport(
			dialer.Dial, b.Address,
		))
	} else if b.Type == "tcp" {
		parentreso := resolver.New(b.Beginning, b.Handler, new(net.Resolver))
		dialer := NewDialer(b.Beginning, b.Handler, parentreso, true)
		reso = oodns.NewClient(b.Handler, dnsovertcp.NewTransport(
			dialer.Dial, b.Address,
		))
	} else if b.Type == "dot" {
		parentreso := resolver.New(b.Beginning, b.Handler, new(net.Resolver))
		dialer := NewDialer(b.Beginning, b.Handler, parentreso, false)
		tlsdialer := NewTLSDialer(b.Beginning, b.Handler, dialer, b.TLSConfig)
		reso = oodns.NewClient(b.Handler, dnsovertcp.NewTransport(
			tlsdialer.DialTLS, b.Address,
		))
	} else if b.Type == "doh" {
		client := NewHTTPClient(b.Beginning, b.Handler, b.TLSConfig, true)
		reso = oodns.NewClient(b.Handler, dnsoverhttps.NewTransport(
			client, b.Address,
		))
	} else {
		err = errors.New("unsupported resolver type")
	}
	if err != nil {
		return nil, err
	}
	reso = resolver.New(b.Beginning, b.Handler, reso)
	return reso, nil
}

// NewDialer creates a new dialer
func NewDialer(
	beginning time.Time, handler model.Handler,
	resolver model.Resolver, includeBytes bool,
) model.Dialer {
	return dialer.New(
		beginning, handler, new(net.Dialer),
		resolver, includeBytes,
	)
}

// NewTLSDialer creates a new TLSDialer
func NewTLSDialer(
	beginning time.Time, handler model.Handler,
	dialer model.Dialer, config *tls.Config,
) model.TLSDialer {
	return tlsdialer.New(beginning, handler, dialer, config)
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(
	beginning time.Time, handler model.Handler,
	dialer model.Dialer, config *tls.Config, includeBody bool,
) http.RoundTripper {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Configure h2 and make sure that the custom TLSConfig we use for dialing
	// is actually compatible with upgrading to h2. (This mainly means we
	// need to make sure we include "h2" in the NextProtos array.) Because
	// http2.ConfigureTransport only returns error when we have already
	// configured http2, it is safe to ignore the return value.
	http2.ConfigureTransport(transport)
	transport.TLSClientConfig.ServerName = config.ServerName
	transport.TLSClientConfig.RootCAs = config.RootCAs
	config = transport.TLSClientConfig
	tlsdialer := NewTLSDialer(beginning, handler, dialer, config)
	transport.Dial = dialer.Dial
	transport.DialContext = dialer.DialContext
	transport.DialTLS = tlsdialer.DialTLS
	return httptransport.New(beginning, handler, transport, includeBody)
}

// NewHTTPClient creates a new HTTP client
func NewHTTPClient(
	beginning time.Time, handler model.Handler,
	config *tls.Config, includeBody bool,
) *http.Client {
	parentreso := resolver.New(beginning, handler, new(net.Resolver))
	dialer := NewDialer(beginning, handler, parentreso, false)
	transport := NewHTTPTransport(
		beginning, handler, dialer, config, true,
	)
	return &http.Client{Transport: transport}
}
