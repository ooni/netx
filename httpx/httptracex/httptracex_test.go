package httptracex

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func makeclient(ts *httptest.Server) (client *http.Client) {
	// we need to replace the default transport with our transport
	// such that we can see all the events
	client = ts.Client()
	transport := client.Transport
	client.Transport = &Tracer{
		RoundTripper: transport,
	}
	return client
}

func handle(w http.ResponseWriter, r *http.Request) {
	// We implement redirection to test for request chains as well
	if r.RequestURI == "/" {
		http.Redirect(w, r, "/antani", 302)
		return
	}
	w.Write([]byte(r.RequestURI))
}

func makeserver() *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(handle))
}

func makeboth() (ts *httptest.Server, client *http.Client) {
	ts = makeserver()
	client = makeclient(ts)
	return
}

// request is the convenience method for performing the request and
// then checking whether the result is okay. The |newrequest| argument
// is because we need to test both with our Tracer and with a
// default Tracer, to make sure we always behave.
func request(
	ts *httptest.Server, client *http.Client,
	urlPath string, expectedBody []byte,
) (*EventsContainer, error) {
	resp, err := client.Get(ts.URL + urlPath)
	if err != nil {
		return httptracex.RequestEventsContainer(req), err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return httptracex.RequestEventsContainer(req), err
	}
	if !bytes.Equal(expectedBody, data) {
		return httptracex.RequestEventsContainer(req), errors.New("The body is not what I expected")
	}
	return httptracex.RequestEventsContainer(req), nil
}

func log(t *testing.T, ec *EventsContainer) {
	data, err := json.MarshalIndent(ec.Events, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))
}

func checkAndIncrementOrDie(
	t *testing.T, ec *EventsContainer, kind EventID, idx *int,
) {
	if ec.Events[*idx].EventID != kind {
		t.Fatal("Unexpected event type")
	}
	*idx++
}

func TestSingleRequestLocalhostSuccess(t *testing.T) {
	ts, client := makeboth()
	defer ts.Close()
	ec, err := request(
		ts, client, "/foobar", []byte("/foobar"),
	)
	if err != nil {
		t.Fatal(err)
	}
	// logging what we see is still useful
	log(t, ec)
	// make sure that the sequence of events is okay
	var index int
	checkAndIncrementOrDie(t, ec, HTTPRequestStart, &index)
	checkAndIncrementOrDie(t, ec, ConnectStart, &index)
	checkAndIncrementOrDie(t, ec, ConnectDone, &index)
	checkAndIncrementOrDie(t, ec, TLSHandshakeStart, &index)
	checkAndIncrementOrDie(t, ec, TLSHandshakeDone, &index)
	for ec.Events[index].EventID == HTTPRequestHeader {
		index++
	}
	checkAndIncrementOrDie(t, ec, HTTPRequestHeadersDone, &index)
	checkAndIncrementOrDie(t, ec, HTTPRequestDone, &index)
	checkAndIncrementOrDie(t, ec, HTTPFirstResponseByte, &index)
	checkAndIncrementOrDie(t, ec, HTTPResponseStatusCode, &index)
	for ec.Events[index].EventID == HTTPResponseHeader {
		index++
	}
	checkAndIncrementOrDie(t, ec, HTTPResponseHeadersDone, &index)
	checkAndIncrementOrDie(t, ec, HTTPResponseDone, &index)
	if len(ec.Events) != index {
		t.Fatal("Unexpected events at the end")
	}
}

func TestSingleRequestLocalhostCertError(t *testing.T) {
	ts, client := makeboth()
	defer ts.Close()
	if strings.Count(ts.URL, "127.0.0.1") != 1 {
		t.Fatal("ts.URL does not contain 127.0.0.1")
	}
	ts.URL = strings.Replace(ts.URL, "127.0.0.1", "localhost", -1)
	ec, err := request(
		ts, client, "/foobar", []byte("/foobar"),
	)
	if err == nil {
		t.Fatal("Expected an error here")
	}
	// logging what we see is still useful
	log(t, ec)
	// make sure that the sequence of events is okay
	var index int
	checkAndIncrementOrDie(t, ec, HTTPRequestStart, &index)
	checkAndIncrementOrDie(t, ec, DNSStart, &index)
	checkAndIncrementOrDie(t, ec, DNSDone, &index)
	checkAndIncrementOrDie(t, ec, ConnectStart, &index)
	checkAndIncrementOrDie(t, ec, ConnectDone, &index)
	checkAndIncrementOrDie(t, ec, TLSHandshakeStart, &index)
	checkAndIncrementOrDie(t, ec, TLSHandshakeDone, &index)
	if len(ec.Events) != index {
		t.Fatal("Unexpected events at the end")
	}
}
