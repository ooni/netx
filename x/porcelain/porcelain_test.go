package porcelain

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/scoreboard"
)

func TestUnitChannelHandlerWriteLateOnChannel(t *testing.T) {
	handler := &channelHandler{
		ch: make(chan model.Measurement),
	}
	var waitgroup sync.WaitGroup
	waitgroup.Add(1)
	go func() {
		time.Sleep(1 * time.Second)
		handler.OnMeasurement(model.Measurement{})
		waitgroup.Done()
	}()
	waitgroup.Wait()
	if handler.lateWrites != 1 {
		t.Fatal("unexpected lateWrites value")
	}
}

func TestIntegrationDNSLookupGood(t *testing.T) {
	ctx := context.Background()
	results, err := DNSLookup(ctx, DNSLookupConfig{
		Hostname: "ooni.io",
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error != nil {
		t.Fatal(results.Error)
	}
	if len(results.Addresses) < 1 {
		t.Fatal("no addresses returned?!")
	}
	if results.TestKeys.Scoreboard == nil {
		t.Fatal("no scoreboard?!")
	}
}

func TestIntegrationDNSLookupCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(), time.Microsecond,
	)
	defer cancel()
	results, err := DNSLookup(ctx, DNSLookupConfig{
		Hostname: "ooni.io",
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error == nil {
		t.Fatal("expected an error here")
	}
	if results.Error.Error() != "generic_timeout_error" {
		t.Fatal("not the error we expected")
	}
	if len(results.Addresses) > 0 {
		t.Fatal("addresses returned?!")
	}
}

func TestIntegrationDNSLookupUnknownDNS(t *testing.T) {
	ctx := context.Background()
	results, err := DNSLookup(ctx, DNSLookupConfig{
		Hostname:      "ooni.io",
		ServerNetwork: "antani",
	})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if results != nil {
		t.Fatal("expected nil results here")
	}
}

func TestIntegrationHTTPDoGood(t *testing.T) {
	ctx := context.Background()
	results, err := HTTPDo(ctx, HTTPDoConfig{
		URL: "http://ooni.io",
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error != nil {
		t.Fatal(results.Error)
	}
	if results.StatusCode != 200 {
		t.Fatal("request failed?!")
	}
	if len(results.Headers) < 1 {
		t.Fatal("no headers?!")
	}
	if len(results.Body) < 1 {
		t.Fatal("no body?!")
	}
	if results.TestKeys.Scoreboard == nil {
		t.Fatal("no scoreboard?!")
	}
}

func TestIntegrationHTTPDoCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(), time.Microsecond,
	)
	defer cancel()
	results, err := HTTPDo(ctx, HTTPDoConfig{
		URL: "http://ooni.io",
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error == nil {
		t.Fatal("expected an error here")
	}
	if results.Error.Error() != "generic_timeout_error" {
		t.Fatal("not the error we expected")
	}
}

func TestIntegrationHTTPDoUnknownDNS(t *testing.T) {
	ctx := context.Background()
	results, err := HTTPDo(ctx, HTTPDoConfig{
		URL:              "http://ooni.io",
		DNSServerNetwork: "antani",
	})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if results != nil {
		t.Fatal("expected nil results here")
	}
}

func TestIntegrationHTTPDoRoundTripError(t *testing.T) {
	ctx := context.Background()
	results, err := HTTPDo(ctx, HTTPDoConfig{
		URL: "http://ooni.io:443", // 443 with http
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error == nil {
		t.Fatal("expected an error here")
	}
}

func TestIntegrationHTTPDoBadURL(t *testing.T) {
	ctx := context.Background()
	results, err := HTTPDo(ctx, HTTPDoConfig{
		URL: "\t",
	})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if results != nil {
		t.Fatal("expected nil results here")
	}
}

func TestIntegrationTLSConnectGood(t *testing.T) {
	ctx := context.Background()
	results, err := TLSConnect(ctx, TLSConnectConfig{
		Address: "ooni.io:443",
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error != nil {
		t.Fatal(results.Error)
	}
	if results.TestKeys.Scoreboard == nil {
		t.Fatal("no scoreboard?!")
	}
}

func TestIntegrationTLSConnectGoodWithDoT(t *testing.T) {
	ctx := context.Background()
	results, err := TLSConnect(ctx, TLSConnectConfig{
		Address:          "ooni.io:443",
		DNSServerNetwork: "dot",
		DNSServerAddress: "9.9.9.9:853",
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error != nil {
		t.Fatal(results.Error)
	}
	if results.TestKeys.Scoreboard == nil {
		t.Fatal("no scoreboard?!")
	}
}

func TestIntegrationTLSConnectCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(), time.Microsecond,
	)
	defer cancel()
	results, err := TLSConnect(ctx, TLSConnectConfig{
		Address: "ooni.io:443",
	})
	if err != nil {
		t.Fatal(err)
	}
	if results.Error == nil {
		t.Fatal("expected an error here")
	}
	if results.Error.Error() != "generic_timeout_error" {
		t.Fatal("not the error we expected")
	}
}

func TestIntegrationTLSConnectUnknownDNS(t *testing.T) {
	ctx := context.Background()
	results, err := TLSConnect(ctx, TLSConnectConfig{
		Address:          "ooni.io:443",
		DNSServerNetwork: "antani",
	})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if results != nil {
		t.Fatal("expected nil results here")
	}
}

func TestMaybeRunTLSChecks(t *testing.T) {
	out := maybeRunTLSChecks(
		context.Background(), handlers.NoHandler,
		&model.XResults{
			Scoreboard: scoreboard.Board{
				TLSHandshakeReset: []scoreboard.TLSHandshakeReset{
					scoreboard.TLSHandshakeReset{
						Domain: "ooni.io",
						RecommendedFollowups: []string{
							"sni_blocking",
						},
					},
				},
			},
		},
	)
	if out == nil {
		t.Fatal("unexpected nil return value")
	}
	if out.Connects == nil {
		t.Fatal("no connects?!")
	}
	if out.HTTPRequests != nil {
		t.Fatal("http requests?!")
	}
	if out.Resolves == nil {
		t.Fatal("no queries?!")
	}
	if out.TLSHandshakes == nil {
		t.Fatal("no TLS handshakes?!")
	}
}
