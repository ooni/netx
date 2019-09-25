# github.com/ooni/netx

[![GoDoc](https://godoc.org/github.com/ooni/netx?status.svg)](https://godoc.org/github.com/ooni/netx) [![Build Status](https://travis-ci.org/ooni/netx.svg?branch=master)](https://travis-ci.org/ooni/netx) [![Coverage Status](https://coveralls.io/repos/github/ooni/netx/badge.svg?branch=master)](https://coveralls.io/github/ooni/netx?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/ooni/netx)](https://goreportcard.com/report/github.com/ooni/netx)

This repository contains `net` and `net/http` extensions for performing
seamless network measurements. It is a meant as PoC for code that I wish to
integrate into [ooni/probe-engine](https://github.com/ooni/probe-engine).

## Build, run tests, run example commands

You need Go >= 1.11. To run tests:

```
go test -v -race ./...
```

To build the example commands:

```
go build -v ./cmd/dnsclient
go build -v ./cmd/httpclient
```

Both commands will provide useful help messages when run with `-help`. When
run without arguments they run against default input suitable to show
at a first glance their functionality.

## Rationale and design

The main design principle implemented here is that we want to perform
measurements on the side with respect to ordinary Go code. Consider for
example the following Go snippet:

```Go
func fetch(client *http.Client, url string) ([]data, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
  return ioutil.ReadAll(resp.Body)
}
```

We want to simplify the implementation of OONI's Web Connectivity
experiment (aka nettest) to be as simple as that. Currently the
test implementation is complex and involves, for each URL, performing
a DNS request, a TCP connect, and an HTTP GET operation. However, if
we collected the data somehow "on the side", the above code snippet
would be enough to implement the bulk of Web Connectivity.

This repository is an attempt at making this enhancement possible, by
implementing measurements on the side. It really boils down to creating
a `httpx.Client` rather than an HTTP client. This enhanced client will
setup automatic measurements. What's more, it contains a public
`HTTPClient *http.Client` field that you can pass to existing code
expecting an instance of `*http.Client`. For example, the following
code snippet

```Go
import (
  "github.com/ooni/netx/handlers"
  "github.com/ooni/netx/httpx"
)

func main() {
  client := httpx.NewClient(handlers.StdoutHandler)
  fetch(client.HTTPClient, "https://ooni.io")
}
```

enhances the above code snippet by printing on `os.Stdout` low
level events including:

* DNS messages (currently only on Unix)
* the result of every Connect, Read, Write, Close operation
* the result of the TLS handshake (including certificates)

The following is a prettyprint of some selected messages
printed on the standard output when running the above code:

```
{"DNSQuery": {
  "ConnID": 2,
  "Message": {"Data": "rdIBAAABAAAAAAAABG9vbmkCaW8AAAEAAQ=="},
  "Time":1836393}}

{"DNSReply": {
  "ConnID": 2,
  "Message": {
    "Data": "rdKBgAABAAEAAAAABG9vbmkCaW8AAAEAAcAMAAEAAQAAASsABGjGDjQ="
  },
  "Time": 104245487}}

{"Write": {
  "ConnID": 1, "Duration": 58541, "Error": null,
  "NumBytes": 80, "Time": 677809240}}

{"TLSHandshake": {"Config": {
    "NextProtos":["h2","http/1.1"],"ServerName":"ooni.io"
  }, "ConnectionState": {
    "CipherSuite": 4866, "NegotiatedProtocol": "h2",
    "NegotiatedProtocolIsMutual": true,
    "PeerCertificates":[
      {"Data":"MIIFXjCCB..."},
      {"Data":"MIIEkjCCA..."}
     ],
     "Version": 772
  },
  "ConnID": 1,
  "Duration": 381079853,
  "Error": null,
  "Time": 677902242}}

  {"HTTPResponseHeadersDone":
    {"Headers": {
      "Age":["13217"],
    },"StatusCode":200,"Time":873592903,"TransactionID":1,}}
```

By passing to `NewClient` a different handler implementing the
`model.Handler` interface, you can store such measurements rather
than printing them on the standard output.

## Using a different DNS

We also want this code to use different kind of DNS transports,
and namely, at leat, DoT and DoH. This is implemented by a method
of the `httpx.Client` object, `ConfigureDNS`.

When you change the DNS transport, you also see events generated
by such transport. For example, if you use DoH, you see the
TLS handshake with the DoH server, HTTP messages, etc.

Please, refer to [cmd/httpclient/main.go](cmd/httpclient/main.go) to
see how this could be used in practice.

## Low level networking

The `httpx` functionality is built on top of low level networking
APIs scattered across the `netx` and `netx/dnsx` packages. The
main object that we expose is a replacement for `net.Dialer`, called
`netx.Dialer`, which implements the following APIs:

```Go
Dial(network, address string) (net.Conn, error)
DialContext(ctx context.Context, network, address string) (net.Conn, error)
DialTLS(network, address string) (conn net.Conn, err error)
```

These are the APIs that several libraries, including `net/http` and
`gorilla/websocket` except an underlying dialer to implement.

You can also call `netx.Dialer.ConfigureDNS` to change the transport to
be used for the DNS, as described above for `httpx.Client`.

The `netx.Dialer.NewResolver` API allows you to get a functional
replacement for a `net.Resolver` object that implements:

```
LookupAddr(ctx context.Context, addr string) (names []string, err error)
LookupCNAME(ctx context.Context, host string) (cname string, err error)
LookupHost(ctx context.Context, hostname string) (addrs []string, err error)
LookupMX(ctx context.Context, name string) ([]*net.MX, error)
LookupNS(ctx context.Context, name string) ([]*net.NS, error)
```

When using the Dialer replacement or the Resolver replacement, the
network level events are being logged as well.

See [cmd/dnsclient/main.go](cmd/dnsclient/main.go) for a more comprehensive
example of how you can use a `net.Resolver` replacement.

## Data model

The data model is in the [model](model) Go package. Please refer to
its documentation for more details.

## Expected integration plan

If this proposal is accepted, I believe we should have this code
live in the `github.com/ooni` namespace as a separate library. IMHO
this code is a lower level of abstraction and
deals with different concerns than `ooni/probe-engine`.
