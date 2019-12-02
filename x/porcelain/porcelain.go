// Package porcelain contains useful high level functionality.
//
// This is the main package used by ooni/probe-engine. The objective
// of this package is to make things simple in probe-engine.
//
// Also, this is currently experimental. So, no API promises here.
package porcelain

import (
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/m-lab/go/rtx"
	"github.com/ooni/netx"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/httpx"
	"github.com/ooni/netx/internal/errwrapper"
	"github.com/ooni/netx/internal/httptransport/tracetripper"
	"github.com/ooni/netx/modelx"
	"github.com/ooni/netx/x/scoreboard"
)

type channelHandler struct {
	ch         chan<- modelx.Measurement
	lateWrites int64
}

func (h *channelHandler) OnMeasurement(m modelx.Measurement) {
	// Implementation note: when we're closing idle connections it
	// may be that they're closed once we have stopped reading
	// therefore (1) we MUST NOT close the channel to signal that
	// we're done BECAUSE THIS IS A LIE and (2) we MUST instead
	// arrange here for non-blocking sends.
	select {
	case h.ch <- m:
	case <-time.After(100 * time.Millisecond):
		atomic.AddInt64(&h.lateWrites, 1)
	}
}

// Results contains the results of every operation that we care
// about, as well as the experimental scoreboard, and information
// on the number of bytes received and sent.
//
// When counting the number of bytes sent and received, we do not
// take into account domain name resolutions performed using the
// system resolver. We estimated that using heuristics with MK but
// we currently don't have a good solution. TODO(bassosimone): this
// can be improved by emitting estimates when we know that we are
// using the system resolver, so we can pick up estimates here.
type Results struct {
	Connects      []*modelx.ConnectEvent
	HTTPRequests  []*modelx.HTTPRoundTripDoneEvent
	Resolves      []*modelx.ResolveDoneEvent
	TLSHandshakes []*modelx.TLSHandshakeDoneEvent

	Scoreboard    *scoreboard.Board
	SentBytes     int64
	ReceivedBytes int64
}

func (r *Results) onMeasurement(m modelx.Measurement) {
	if m.Connect != nil {
		r.Connects = append(r.Connects, m.Connect)
	}
	if m.HTTPRoundTripDone != nil {
		r.HTTPRequests = append(r.HTTPRequests, m.HTTPRoundTripDone)
	}
	if m.ResolveDone != nil {
		r.Resolves = append(r.Resolves, m.ResolveDone)
	}
	if m.TLSHandshakeDone != nil {
		r.TLSHandshakes = append(r.TLSHandshakes, m.TLSHandshakeDone)
	}
	if m.Read != nil {
		r.ReceivedBytes += m.Read.NumBytes // overflow unlikely
	}
	if m.Write != nil {
		r.SentBytes += m.Write.NumBytes // overflow unlikely
	}
}

func (r *Results) collect(
	output <-chan modelx.Measurement,
	handler modelx.Handler,
	main func(),
) {
	if handler == nil {
		handler = handlers.NoHandler
	}
	done := make(chan interface{})
	go func() {
		defer close(done)
		main()
	}()
	for {
		select {
		case m := <-output:
			handler.OnMeasurement(m)
			r.onMeasurement(m)
		case <-done:
			return
		}
	}
}

type dnsFallback struct {
	network, address string
}

func configureDNS(seed int64, network, address string) (modelx.DNSResolver, error) {
	resolver, err := netx.NewResolver(handlers.NoHandler, network, address)
	if err != nil {
		return nil, err
	}
	fallbacks := []dnsFallback{
		dnsFallback{
			network: "doh",
			address: "https://cloudflare-dns.com/dns-query",
		},
		dnsFallback{
			network: "doh",
			address: "https://dns.google/dns-query",
		},
		dnsFallback{
			network: "dot",
			address: "8.8.8.8:853",
		},
		dnsFallback{
			network: "dot",
			address: "8.8.4.4:853",
		},
		dnsFallback{
			network: "dot",
			address: "1.1.1.1:853",
		},
		dnsFallback{
			network: "dot",
			address: "9.9.9.9:853",
		},
	}
	random := rand.New(rand.NewSource(seed))
	random.Shuffle(len(fallbacks), func(i, j int) {
		fallbacks[i], fallbacks[j] = fallbacks[j], fallbacks[i]
	})
	var configured int
	for i := 0; configured < 2 && i < len(fallbacks); i++ {
		if fallbacks[i].network == network {
			continue
		}
		var fallback modelx.DNSResolver
		fallback, err = netx.NewResolver(
			handlers.NoHandler, fallbacks[i].network,
			fallbacks[i].address,
		)
		rtx.PanicOnError(err, "porcelain: invalid fallbacks table")
		resolver = netx.ChainResolvers(resolver, fallback)
		configured++
	}
	return resolver, nil
}

// DNSLookupConfig contains DNSLookup settings.
type DNSLookupConfig struct {
	Handler       modelx.Handler
	Hostname      string
	ServerAddress string
	ServerNetwork string
}

// DNSLookupResults contains the results of a DNSLookup
type DNSLookupResults struct {
	TestKeys  Results
	Addresses []string
	Error     error
}

// DNSLookup performs a DNS lookup.
func DNSLookup(
	ctx context.Context, config DNSLookupConfig,
) (*DNSLookupResults, error) {
	channel := make(chan modelx.Measurement)
	root := &modelx.MeasurementRoot{
		Beginning: time.Now(),
		Handler: &channelHandler{
			ch: channel,
		},
	}
	ctx = modelx.WithMeasurementRoot(ctx, root)
	resolver, err := netx.NewResolver(
		handlers.NoHandler,
		config.ServerNetwork,
		config.ServerAddress,
	)
	if err != nil {
		return nil, err
	}
	var (
		mu      sync.Mutex
		results = new(DNSLookupResults)
	)
	results.TestKeys.collect(channel, config.Handler, func() {
		addrs, err := resolver.LookupHost(ctx, config.Hostname)
		mu.Lock()
		defer mu.Unlock()
		results.Addresses, results.Error = addrs, err
	})
	results.TestKeys.Scoreboard = &root.X.Scoreboard
	return results, nil
}

// HTTPDoConfig contains HTTPDo settings.
type HTTPDoConfig struct {
	Accept           string
	AcceptLanguage   string
	Body             []byte
	DNSServerAddress string
	DNSServerNetwork string
	Handler          modelx.Handler
	Method           string
	ProxyFunc        func(*http.Request) (*url.URL, error)
	URL              string
	UserAgent        string

	// MaxEventsBodySnapSize controls the snap size that
	// we're using for bodies returned as modelx.Measurement.
	//
	// Same rules as modelx.MeasurementRoot.MaxBodySnapSize.
	MaxEventsBodySnapSize int64

	// MaxResponseBodySnapSize controls the snap size that
	// we're using for the HTTPDoResults.BodySnap.
	//
	// Same rules as modelx.MeasurementRoot.MaxBodySnapSize.
	MaxResponseBodySnapSize int64
}

// HTTPDoResults contains the results of a HTTPDo
type HTTPDoResults struct {
	TestKeys            Results
	StatusCode          int64
	Headers             http.Header
	BodySnap            []byte
	Error               error
	SNIBlockingFollowup *modelx.XSNIBlockingFollowup
}

// HTTPDo performs a HTTP request
func HTTPDo(
	origCtx context.Context, config HTTPDoConfig,
) (*HTTPDoResults, error) {
	channel := make(chan modelx.Measurement)
	// TODO(bassosimone): tell client to use specific CA bundle?
	root := &modelx.MeasurementRoot{
		Beginning: time.Now(),
		Handler: &channelHandler{
			ch: channel,
		},
		MaxBodySnapSize: config.MaxEventsBodySnapSize,
	}
	ctx := modelx.WithMeasurementRoot(origCtx, root)
	client := httpx.NewClientWithProxyFunc(handlers.NoHandler, config.ProxyFunc)
	resolver, err := configureDNS(
		time.Now().UnixNano(),
		config.DNSServerNetwork,
		config.DNSServerAddress,
	)
	if err != nil {
		return nil, err
	}
	client.SetResolver(resolver)
	// TODO(bassosimone): implement sending body
	req, err := http.NewRequest(config.Method, config.URL, nil)
	if err != nil {
		return nil, err
	}
	if config.Accept != "" {
		req.Header.Set("Accept", config.Accept)
	}
	if config.AcceptLanguage != "" {
		req.Header.Set("Accept-Language", config.AcceptLanguage)
	}
	req.Header.Set("User-Agent", config.UserAgent)
	req = req.WithContext(ctx)
	var (
		mu      sync.Mutex
		results = new(HTTPDoResults)
	)
	results.TestKeys.collect(channel, config.Handler, func() {
		defer client.HTTPClient.CloseIdleConnections()
		resp, err := client.HTTPClient.Do(req)
		if err != nil {
			mu.Lock()
			results.Error = err
			mu.Unlock()
			return
		}
		mu.Lock()
		results.StatusCode = int64(resp.StatusCode)
		results.Headers = resp.Header
		mu.Unlock()
		defer resp.Body.Close()
		reader := io.LimitReader(
			resp.Body, tracetripper.ComputeBodySnapSize(
				config.MaxResponseBodySnapSize,
			),
		)
		data, err := ioutil.ReadAll(reader)
		mu.Lock()
		results.BodySnap, results.Error = data, err
		mu.Unlock()
	})
	// For safety wrap the error as "http_round_trip" but this
	// will only be used if the error chain does not contain any
	// other major operation failure. See modelx.ErrWrapper.
	results.Error = errwrapper.SafeErrWrapperBuilder{
		Error:     results.Error,
		Operation: "http_round_trip",
	}.MaybeBuild()
	results.TestKeys.Scoreboard = &root.X.Scoreboard
	results.SNIBlockingFollowup = maybeRunTLSChecks(
		origCtx, config.Handler, &root.X,
	)
	return results, nil
}

// TLSConnectConfig contains TLSConnect settings.
type TLSConnectConfig struct {
	Address          string
	DNSServerAddress string
	DNSServerNetwork string
	Handler          modelx.Handler
	SNI              string
}

// TLSConnectResults contains the results of a TLSConnect
type TLSConnectResults struct {
	TestKeys Results
	Error    error
}

// TLSConnect performs a TLS connect.
func TLSConnect(
	ctx context.Context, config TLSConnectConfig,
) (*TLSConnectResults, error) {
	channel := make(chan modelx.Measurement)
	root := &modelx.MeasurementRoot{
		Beginning: time.Now(),
		Handler: &channelHandler{
			ch: channel,
		},
	}
	ctx = modelx.WithMeasurementRoot(ctx, root)
	dialer := netx.NewDialer(handlers.NoHandler)
	// TODO(bassosimone): tell dialer to use specific CA bundle?
	resolver, err := configureDNS(
		time.Now().UnixNano(),
		config.DNSServerNetwork,
		config.DNSServerAddress,
	)
	if err != nil {
		return nil, err
	}
	dialer.SetResolver(resolver)
	// TODO(bassosimone): can this call really fail?
	dialer.ForceSpecificSNI(config.SNI)
	var (
		mu      sync.Mutex
		results = new(TLSConnectResults)
	)
	results.TestKeys.collect(channel, config.Handler, func() {
		conn, err := dialer.DialTLSContext(ctx, "tcp", config.Address)
		if conn != nil {
			defer conn.Close()
		}
		mu.Lock()
		defer mu.Unlock()
		results.Error = err
	})
	results.TestKeys.Scoreboard = &root.X.Scoreboard
	return results, nil
}

func maybeRunTLSChecks(
	ctx context.Context, handler modelx.Handler, x *modelx.XResults,
) (out *modelx.XSNIBlockingFollowup) {
	for _, ev := range x.Scoreboard.TLSHandshakeReset {
		for _, followup := range ev.RecommendedFollowups {
			if followup == "sni_blocking" {
				out = sniBlockingFollowup(ctx, handler, ev.Domain)
				break
			}
		}
	}
	return
}

// TODO(bassosimone): we should make this configurable
const sniBlockingHelper = "example.com:443"

func sniBlockingFollowup(
	ctx context.Context, handler modelx.Handler, domain string,
) (out *modelx.XSNIBlockingFollowup) {
	config := TLSConnectConfig{
		Address: sniBlockingHelper,
		Handler: handler,
		SNI:     domain,
	}
	measurements, err := TLSConnect(ctx, config)
	if err == nil {
		out = &modelx.XSNIBlockingFollowup{
			Connects:      measurements.TestKeys.Connects,
			HTTPRequests:  measurements.TestKeys.HTTPRequests,
			Resolves:      measurements.TestKeys.Resolves,
			TLSHandshakes: measurements.TestKeys.TLSHandshakes,
		}
	}
	return
}
