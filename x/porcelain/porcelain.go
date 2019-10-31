// Package porcelain contains useful high level functionality.
//
// This is the main package used by ooni/probe-engine. The objective
// of this package is to make things simple in probe-engine.
//
// Also, this is currently experimental. So, no API promises here.
package porcelain

import (
	"crypto/x509"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/ooni/netx"
	"github.com/ooni/netx/handlers"
	"github.com/ooni/netx/httpx"
	"github.com/ooni/netx/internal/errwrapper"
	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/scoreboard"
)

// NewHTTPRequest is like http.NewRequest except that it also adds
// to such request a configured MeasurementRoot.
func NewHTTPRequest(method, URL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, URL, body)
	if err == nil {
		root := &model.MeasurementRoot{
			Beginning: time.Now(),
			Handler:   handlers.NoHandler,
		}
		ctx := model.WithMeasurementRoot(req.Context(), root)
		req = req.WithContext(ctx)
	}
	return req, err
}

// RequestMeasurementRoot returns the MeasurementRoot that has been
// configured into the context of a request, or nil.
func RequestMeasurementRoot(req *http.Request) *model.MeasurementRoot {
	return model.ContextMeasurementRoot(req.Context())
}

// HTTPRequest contains the request summary. This is structured so
// that it's easy to generate OONI events.
type HTTPRequest struct {
	Method  string
	URL     string
	Headers http.Header
	Body    string
}

// HTTPResponse contains the response summary. This is structured so
// that it's easy to generate OONI events.
type HTTPResponse struct {
	StatusCode int64
	Headers    http.Header
	Body       string
}

// HTTPTransaction contains information on an HTTP transaction, i.e.
// on an HTTP round trip plus the response body. This is structured so
// that it's easy to generate OONI events.
type HTTPTransaction struct {
	// DurationSinceBeginning is the number of nanoseconds since
	// the time configured as the "zero" time.
	DurationSinceBeginning time.Duration

	// Error contains the overall transaction error.
	Error error

	// Request contains information on the request.
	Request HTTPRequest

	// Response contains information on the response.
	Response HTTPResponse

	// TransactionID is the identifier of this transaction
	TransactionID int64
}

type getHandler struct {
	connects     []*model.ConnectEvent
	handler      model.Handler
	handshakes   []*model.TLSHandshakeDoneEvent
	lastTxID     int64
	mu           sync.Mutex
	resolves     []*model.ResolveDoneEvent
	transactions []*HTTPTransaction
}

func (h *getHandler) OnMeasurement(m model.Measurement) {
	defer h.handler.OnMeasurement(m)
	h.mu.Lock()
	defer h.mu.Unlock()
	// Implementation details re: lastTxID:
	//
	// 1. the round trip should always be the last event but
	// I've decided to make the code more robust
	//
	// 2. the TLS handshake should have a transaction ID since
	// it's run by the net/http code, but again robustness
	if m.ResolveDone != nil {
		h.resolves = append(h.resolves, m.ResolveDone)
		h.lastTxID = m.ResolveDone.TransactionID
	}
	if m.Connect != nil {
		h.connects = append(h.connects, m.Connect)
		h.lastTxID = m.Connect.TransactionID
	}
	if m.TLSHandshakeDone != nil {
		h.handshakes = append(h.handshakes, m.TLSHandshakeDone)
		if m.TLSHandshakeDone.TransactionID != 0 {
			h.lastTxID = m.TLSHandshakeDone.TransactionID
		}
	}
	if m.HTTPRoundTripDone != nil {
		rtinfo := m.HTTPRoundTripDone
		h.lastTxID = rtinfo.TransactionID
		// We're saving the RedirectBody as body, which is correct for
		// all requests in the chain except the last one. We will change
		// the body later so it's always correct.
		h.transactions = append(h.transactions, &HTTPTransaction{
			DurationSinceBeginning: rtinfo.DurationSinceBeginning,
			Error:                  rtinfo.Error,
			Request: HTTPRequest{
				Method:  rtinfo.RequestMethod,
				URL:     rtinfo.RequestURL,
				Headers: rtinfo.RequestHeaders,
				Body:    string(rtinfo.RequestBodySnap),
			},
			Response: HTTPResponse{
				StatusCode: rtinfo.StatusCode,
				Headers:    rtinfo.Headers,
				Body:       string(rtinfo.BodySnap),
			},
			TransactionID: rtinfo.TransactionID,
		})
	}
}

// HTTPMeasurements contains all the measurements performed
// during a full chain of GET redirects.
type HTTPMeasurements struct {
	Resolves   []*model.ResolveDoneEvent
	Connects   []*model.ConnectEvent
	Handshakes []*model.TLSHandshakeDoneEvent
	Requests   []*HTTPTransaction
	Scoreboard *scoreboard.Board
}

// Get fetches the resources at URL using the specified User-Agent
// string, using the specified events handler, and HTTPX client.
//
// This function will return the list of events seen, divided by
// operation: RESOLVE, CONNECT, REQUEST, etc.
func Get(
	handler model.Handler, client *httpx.Client, URL, UserAgent string,
) (*HTTPMeasurements, error) {
	req, err := NewHTTPRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", UserAgent)
	root := RequestMeasurementRoot(req)
	gethandler := &getHandler{handler: handler}
	root.Handler = gethandler
	measurements := new(HTTPMeasurements)
	resp, err := client.HTTPClient.Do(req)
	err = errwrapper.SafeErrWrapperBuilder{
		Error:         err,
		TransactionID: gethandler.lastTxID,
	}.MaybeBuild()
	var body []byte
	if err == nil {
		// Important here to override the outer `err` rather
		// than defining a new `err` in this small scope
		body, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
	gethandler.mu.Lock() // probably superfluous
	defer gethandler.mu.Unlock()
	measurements.Resolves = gethandler.resolves
	measurements.Connects = gethandler.connects
	measurements.Handshakes = gethandler.handshakes
	measurements.Requests = gethandler.transactions
	total := len(measurements.Requests)
	if total >= 1 {
		// We should always have a transaction but I've decided
		// writing robust code here was better
		measurements.Requests[total-1].Error = err
		// As mentioned above, make sure the last transaction in
		// the chain gets the correct body. It has the redirect body
		// in it, which is not set for non-redirects.
		measurements.Requests[total-1].Response.Body = string(body)
	}
	measurements.Scoreboard = &root.X.Scoreboard
	return measurements, err
}

// NewHTTPXClient returns a new HTTPX client
func NewHTTPXClient() *httpx.Client {
	return httpx.NewClient(handlers.NoHandler)
}

// TLSMeasurements contains all the measurements performed
// during a TLS connection attempt with a server.
type TLSMeasurements struct {
	Resolve            *model.ResolveDoneEvent
	Connects           []*model.ConnectEvent
	Error              error
	ClientSNI          string
	CipherSuite        uint16
	NegotiatedProtocol string
	PeerCertificates   []model.X509Certificate
	Version            uint16
}

type tlsHandler struct {
	resolve   *model.ResolveDoneEvent
	connects  []*model.ConnectEvent
	handler   model.Handler
	handshake *model.TLSHandshakeDoneEvent
	mu        sync.Mutex
}

func (h *tlsHandler) OnMeasurement(m model.Measurement) {
	defer h.handler.OnMeasurement(m)
	h.mu.Lock()
	defer h.mu.Unlock()
	// TODO(bassosimone): do I need to write paranoid code here and
	// handle the case of multiple resolve, handshake events?
	if m.ResolveDone != nil {
		h.resolve = m.ResolveDone
	}
	if m.Connect != nil {
		h.connects = append(h.connects, m.Connect)
	}
	if m.TLSHandshakeDone != nil {
		h.handshake = m.TLSHandshakeDone
	}
}

// TLSConnect connects to address and performs a TLS handshake
// using the provided SNI name. This is the base building block
// of an experiment for measuring TLS SNI based blocking.
func TLSConnect(
	handler model.Handler, address string, sni string,
) (*TLSMeasurements, error) {
	tlshandler := &tlsHandler{handler: handler}
	dialer := netx.NewDialer(tlshandler)
	dialer.ForceSpecificSNI(sni)
	conn, err := dialer.DialTLS("tcp", address)
	if conn != nil {
		defer conn.Close()
	}
	tlshandler.mu.Lock()
	defer tlshandler.mu.Unlock()
	measurements := &TLSMeasurements{
		Resolve:   tlshandler.resolve,
		Connects:  tlshandler.connects,
		Error:     err,
		ClientSNI: sni,
	}
	if err != nil {
		// Catch the case where we failed because the SNI is not valid
		// and move the certificate and the SNI out of the error. Remove
		// also the original error because it makes the JSON noisy and
		// we have extracted all the information we needed anyway.
		var wrapper *model.ErrWrapper
		if errors.As(err, &wrapper) && wrapper.Failure == "ssl_invalid_hostname" {
			var x509HostnameError x509.HostnameError
			if errors.As(wrapper.WrappedErr, &x509HostnameError) {
				wrapper.WrappedErr = nil
				certs := []*x509.Certificate{x509HostnameError.Certificate}
				measurements.PeerCertificates = model.SimplifyCerts(certs)
				// TODO(bassosimone): is it too paranoid to make sure
				// that x509HostnameError.Host == sni?
			}
		}
	} else if tlshandler.handshake != nil {
		state := tlshandler.handshake.ConnectionState
		measurements.CipherSuite = state.CipherSuite
		measurements.NegotiatedProtocol = state.NegotiatedProtocol
		measurements.PeerCertificates = state.PeerCertificates
		measurements.Version = state.Version
	}
	return measurements, err
}
