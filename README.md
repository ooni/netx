# github.com/ooni/netx

⚠️: As of 2020-03-06, netx has been merged into ooni/probe-engine. You should
import github.com/ooni/probe-engine/netx rather than github.com/ooni/netx. We
thought it was easier to keep netx and probe-engine separate, but it turned
out this was increasingly slowing us down, so we decided to merge them.

![Build Status](https://github.com/ooni/netx/workflows/Build/badge.svg) [![Coverage Status](https://coveralls.io/repos/github/ooni/netx/badge.svg?branch=master)](https://coveralls.io/github/ooni/netx?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/ooni/netx)](https://goreportcard.com/report/github.com/ooni/netx)

OONI extensions to the `net` and `net/http` packages. This code is
used by `ooni/probe-engine` as a low level library to collect
network, DNS, and HTTP events occurring during OONI measurements.

## API documentation

This library contains replacements for commonly used standard library
interfaces that facilitate seamless network measurements. By using
such replacements, as opposed to standard library interfaces, we can:

* save the timing of HTTP events (e.g. received response headers)
* save the timing and result of every Connect, Read, Write, Close operation
* save the timing and result of the TLS handshake (including certificates)

By default, this library uses the system resolver. In addition, it
is possible to configure alternative DNS transports and remote
servers. We support DNS over UDP, DNS over TCP, DNS over TLS (DoT),
and DNS over HTTPS (DoH). When using an alternative transport, we
are also able to intercept and save DNS messages, as well as any
other interaction with the remote server (e.g., the result of the
TLS handshake for DoT and DoH).

### github.com/ooni/netx/modelx

[![GoDoc](https://godoc.org/github.com/ooni/netx/modelx?status.svg)](
https://godoc.org/github.com/ooni/netx/modelx)

The base package, that defines everything that other packages
will use, and chiefly the measurement model.

### github.com/ooni/netx/httpx

[![GoDoc](https://godoc.org/github.com/ooni/netx/httpx?status.svg)](
https://godoc.org/github.com/ooni/netx/httpx)

Implements a `http.Client` replacement that saves the timing and
results of HTTP and network events.

### github.com/ooni/netx

[![GoDoc](https://godoc.org/github.com/ooni/netx?status.svg)](
https://godoc.org/github.com/ooni/netx)

Implements `net.Dialer` and `net.Resolver` replacements that saves the
timing and the details of network events.

### Other packages

There are other utility and internal packages. Their documentation
is reachable from [the netx online documentation](
https://godoc.org/github.com/ooni/netx#pkg-subdirectories).

## Build, run tests, run example commands

You need Go >= 1.13. We use Go modules.

To run tests:

```
GO111MODULE=on go test -v -race ./...
```

To build the example commands:

```
GO111MODULE=on go build -v ./cmd/...
```

All commands will provide terse help messages when run with `-help`. When
run without arguments they run against default input suitable to show
at a first glance their functionality.
