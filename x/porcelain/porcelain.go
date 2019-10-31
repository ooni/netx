// Package porcelain contains useful high level functionality.
//
// This is the main package used by ooni/probe-engine. The objective
// of this package is to make things simple in probe-engine.
//
// Also, this is currently experimental. So, no API promises here.
package porcelain

import (
	"context"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/netx"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/httpx"
	"github.com/ooni/netx/internal/errwrapper"
	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/scoreboard"
)

type channelHandler struct {
	ch         chan<- model.Measurement
	lateWrites int64
}

func (h *channelHandler) OnMeasurement(m model.Measurement) {
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

// Results contains the results of any operation.
type Results struct {
	Connects      []*model.ConnectEvent
	HTTPRequests  []*model.HTTPRoundTripDoneEvent
	Queries       []*model.ResolveDoneEvent
	Scoreboard    *scoreboard.Board
	TLSHandshakes []*model.TLSHandshakeDoneEvent
}

func (r *Results) onMeasurement(m model.Measurement) {
	if m.Connect != nil {
		r.Connects = append(r.Connects, m.Connect)
	}
	if m.HTTPRoundTripDone != nil {
		r.HTTPRequests = append(r.HTTPRequests, m.HTTPRoundTripDone)
	}
	if m.ResolveDone != nil {
		r.Queries = append(r.Queries, m.ResolveDone)
	}
	if m.TLSHandshakeDone != nil {
		r.TLSHandshakes = append(r.TLSHandshakes, m.TLSHandshakeDone)
	}
}

func (r *Results) collect(
	output <-chan model.Measurement,
	handler model.Handler,
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

// DNSLookupConfig contains DNSLookup settings.
type DNSLookupConfig struct {
	Handler       model.Handler
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
	channel := make(chan model.Measurement)
	// TODO(bassosimone): tell DoH to use specific CA bundle?
	root := &model.MeasurementRoot{
		Beginning: time.Now(),
		Handler: &channelHandler{
			ch: channel,
		},
	}
	ctx = model.WithMeasurementRoot(ctx, root)
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
	// TODO(bassosimone): tell DoH to close idle connections?
	return results, nil
}

// HTTPDoConfig contains HTTPDo settings.
type HTTPDoConfig struct {
	Body             []byte
	DNSServerAddress string
	DNSServerNetwork string
	Handler          model.Handler
	Method           string
	URL              string
	UserAgent        string
}

// HTTPDoResults contains the results of a HTTPDo
type HTTPDoResults struct {
	TestKeys   Results
	StatusCode int64
	Headers    http.Header
	Body       []byte
	Error      error
}

// HTTPDo performs a HTTP request
func HTTPDo(
	ctx context.Context, config HTTPDoConfig,
) (*HTTPDoResults, error) {
	channel := make(chan model.Measurement)
	// TODO(bassosimone): tell client to use specific CA bundle?
	root := &model.MeasurementRoot{
		Beginning: time.Now(),
		Handler: &channelHandler{
			ch: channel,
		},
	}
	ctx = model.WithMeasurementRoot(ctx, root)
	client := httpx.NewClient(handlers.NoHandler)
	err := client.ConfigureDNS(
		config.DNSServerNetwork, config.DNSServerAddress,
	)
	if err != nil {
		return nil, err
	}
	// TODO(bassosimone): implement sending body
	req, err := http.NewRequest(config.Method, config.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", config.UserAgent)
	req = req.WithContext(ctx)
	var (
		mu      sync.Mutex
		results = new(HTTPDoResults)
	)
	results.TestKeys.collect(channel, config.Handler, func() {
		defer client.HTTPClient.CloseIdleConnections()
		// TODO(bassosimone): tell DoH to close idle connections?
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
		data, err := ioutil.ReadAll(resp.Body)
		mu.Lock()
		results.Body, results.Error = data, err
		mu.Unlock()
	})
	results.Error = errwrapper.SafeErrWrapperBuilder{
		Error: results.Error,
	}.MaybeBuild()
	results.TestKeys.Scoreboard = &root.X.Scoreboard
	return results, nil
}

// TLSConnectConfig contains TLSConnect settings.
type TLSConnectConfig struct {
	Address          string
	DNSServerAddress string
	DNSServerNetwork string
	Handler          model.Handler
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
	channel := make(chan model.Measurement)
	root := &model.MeasurementRoot{
		Beginning: time.Now(),
		Handler: &channelHandler{
			ch: channel,
		},
	}
	ctx = model.WithMeasurementRoot(ctx, root)
	dialer := netx.NewDialer(handlers.NoHandler)
	// TODO(bassosimone): tell dialer to use specific CA bundle?
	err := dialer.ConfigureDNS(
		config.DNSServerNetwork, config.DNSServerAddress,
	)
	if err != nil {
		return nil, err
	}
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
