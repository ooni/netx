# github.com/bassosimone/netx

[![GoDoc](https://godoc.org/github.com/bassosimone/netx?status.svg)](https://godoc.org/github.com/bassosimone/netx) [![Build Status](https://travis-ci.org/bassosimone/netx.svg?branch=master)](https://travis-ci.org/bassosimone/netx) [![Coverage Status](https://coveralls.io/repos/github/bassosimone/netx/badge.svg?branch=master)](https://coveralls.io/github/bassosimone/netx?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/bassosimone/netx)](https://goreportcard.com/report/github.com/bassosimone/netx)

This repository contains `net` and `net/http` extensions for performing
seamless network measurements. It is a meant as PoC for code that I wish to
integrate into [ooni/probe-engine](https://github.com/ooni/probe-engine).

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
  "github.com/bassosimone/netx/handlers"
  "github.com/bassosimone/netx/httpx"
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