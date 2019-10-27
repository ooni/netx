# OONI Network Extensions

| Author       | Simone Basso |
|--------------|--------------|
| Last-Updated | 2019-10-27   |
| Status       | approved     |

## Introduction

OONI experiments send and/or receive network traffic to
determine if there is blocking. We want the implementation
of OONI experiments to be as simple as possible.

At the same time, we want an experiment to collect as much
low-level data as possible. For eample, we want to know
whether and when the TLS handshake completed; what certificates
were provided by the server; what TLS version was selected;
and so forth. These bits of information are very useful
to analyze a measurement and better classify it.

We also want to be able to change some configuration properties
and repeat the measurement; e.g., we may want to configure DNS
over HTTPS (DoH) and then attempt to fetch again an URL. Or
we may want to force TLS to use a specific SNI.

In this document we design a Go library that solves all the
above problems by exposing to the user an API with simple replacements
for standard Go interfaces, e.g. `http.RoundTripper`.

## Rationale

As we observed [in a recent ooni/probe-engine issue](
https://github.com/ooni/probe-engine/issues/13), every
experiment consists of two separate phases:

1. measurement gathering

2. measurement analysis

During measurement gathering, we perform specific actions
that cause network data to be sent and/or received. During
measurement analysis, we process the measurement on the
device. For some experiments (e.g., Web Connectivity), this
second phase also entails contacting OONI backend services
that provide data useful to complete the analysis.

In [Measurement Kit](https//github.com/measurement-kit/measurement-it),
we implement measurement gathering by combining _nettest
templates_. These are special APIs that perform a single
low-level action (e.g. connecting to a TCP endpoint, resolving
a domain name). So, for example, Web Connectivity's
measurement gathering is obtained by combining the DNS
template, the TCP template, and the HTTP template.

This approach based on combining low-level test helpers
has two problems. First, the implementation of an
experiment is rather low level, because you need to
invoke the test helpers in sequence, to populate the
measurement result object. Second, the test helpers API is likely
to eventually change when new measurement techniques
are added to the measurement engine.

Because Go has powerful interfaces, we propose in this
document to use an alternative approach where we provide
OONI-measurements-aware replacements for Go standard
library interfaces, e.g., `http.RoundTripper`.

This repository is separate from `ooni/probe-engine`
because they solve different problems. Here we provide
replacements for standard Go library interfaces that
allow us to perform measurements. In `probe-engine` we
implement OONI tests and clients for OONI backend
services. Putting all the code into the same repository
would have put too many concerns into the same repo.

## Design

We want to provide moral replacements for the following
interfaces in the Go standard library:

1. `http.RoundTripper`

2. `http.Client`

3. `net.Dialer`

4. `net.Resolver`

Where possible (e.g. for `http.RoundTripper`) we will
provide structures implementing the interface. Where
instead this is not possible (e.g. for `net.Dialer`) we
will provide structures implementing methods that are
compatible with the originals. For example, in the
case of `net.Dialer`, we will provide compatible
functions, such as `Dial`, `DialContext`, and `DialTLS`.

This make it possible to use our `net.Dialer`
replacement with other libraries. Both `http.Transport`
and `gorilla/websocket`'s `websocket.Dialer` have 
functions like `Dial` and `DialContext` that can be
overriden. Therefore, we will be able to use our
replacements to collect measurements.

There will be a mechanism for gathering such low
level measurements as they occur, for logging
and/or storing purposes.

## Implementation

The actual implementation must follow this spec. It may include more
methods or interfaces. The exact structure of measurements events
is left unspecified, as they are likely to change. That said, we will
be careful to not remove existing fields and/or change the meaning
of existing fields unless that is necessary.

### The github.com/ooni/netx/model package

This package will contain the definition of low-level
events. We are interested in knowing the following:

1. the timing and result of each I/O operation.

2. the timing of HTTP events occurring during the
lifecycle of an HTTP request.

3. the timing and result of the TLS handshake including
the negotiated TLS version and other details such as
what certificates the server has provided.

4. DNS events, e.g. queries and replies, generated
as part of using DoT and DoH.

Hence, this package should define measurement events
representing each of the above. We will use types
as close as possible to standard Go types, e.g. we
will use `time.Duration` to represent the elapsed
time since a specific "zero", because this will allow
for easy further processing of events.

This package will also contain the definition of the
following interface:

```Go
type Handler interface {
    OnMeasurement(Measurement)
}
```

Every replacement that we write will call the
`OnMeasurement` method of the handler wherever
there is a measurement event.

In turn, the `Measurement` event will be defined
as follows:

```Go
type Measurement struct {
    Read  *ReadEvent
    Write *WriteEvent

    // more similar events ...
}
```

That is, it will contain a pointer for every event
that we support. The events processing code will
check what pointer or pointers are not `nil` to
known which event or events have occurred.

For every detail regarding the structure of the
events, we defer to the current docs.

### The github.com/ooni/netx/httpx package

This package will contain HTTP extensions. The core
structure that we will provide is as follows:

```Go
type Client struct {
  HTTPClient *http.Client
  Transport  *Transport
}
```

Client code is expected to create a `*Client` instance
using the `NewClient` constructor, configure it, and
then pass to code that needs it `HTTPClient` as the real
`*http.Client` instance.

To configure our `*Client` instance, one could use the
`ConfigureDNS`, `SetCABundle` and `ForceSpecificSNI`
methods. They should all be called before using the
`HTTPClient` field, as they'll not be goroutine safe.

```Go
func (c *Client) SetCABundle(path string) error
```

The `SetCABundle` forces using a specific CA bundle,
which is what we already do in OONI Probe.

```Go
func (c *Client) ForceSpecificSNI(sni string) error
```

The `ForceSpecificSNI` forces the TLS code to use a
specific SNI when connecting. This allows us to check
whether there is SNI-based blocking.

```Go
func (c *Client) ConfigureDNS(network, address string) error
```

The `ConfigureDNS` method will behave exactly like the
`ConfigureDNS` method of `netx.Resolver` (see below).

```Go
func (c *Client) SetProxyFunc(f func(*Request) (*url.URL, error) error
```

The `SetProxyFunc` will allow us to configure
a specific proxy. This is useful to have precise
measurements of requests over, say, Psiphon.

Lastly, one will construct an `http.Client` using:

```Go
func NewClient(handler model.Handler) *Client
```

The `handler` shall point to a structure implementing the
`model.Handler` interface. Also, this constructor will
automatically record the current time as the "zero" time
used to compute the `Time` field of every event.

Passing a handler to an HTTP client is fine for logging
but less optimal for recording events caused by each HTTP
round trip. To this end, it may be more convenent to use
_context rooted measurements_, instead:

```Go
import (
    "net/http"
    "time"

    "github.com/ooni/netx/handlers"
    "github.com/ooni/netx/httpx"
)

var client = httpx.NewClient(handlers.NoHandler).HTTPClient

func fetchURL(URL string) (*http.Response, error) {
    req, err := http.NewRequest("GET", URL, nil)
    if err != nil {
        return nil, err
    }
    // The following code will update the request context and cause
    // events to be delivered to the specified handler.
    root := &model.MeasurementRoot{
        Beginning: time.Now(),
        Handler:   handlers.StdoutHandler, // your handler here
    }
    ctx := req.Context()
    ctx = model.WithMeasurementRoot(ctx, root)
    req = req.WithContext(ctx)
    return client.Do(req)
}
```

Of course, you should probably use your handler there.

### The github.com/ooni/netx package

This package will contain a replacement for `net.Dialer`,
called `netx.Dialer`, that exposes the following API:

```Go
func (d *Dialer) Dial(network, address string) (net.Conn, error)
```

```Go
func (d *Dialer) DialContext(
    ctx context.Context, network, address string,
) (net.Conn, error)
```

```Go
func (d *Dialer) DialTLS(network, address string) (conn net.Conn, err error)
```

These three functions will behave exactly as the same
functions in the Go standard library, except that they
will perform measurements. A `Dialer` replacement will be
constructed like:

```Go
func NewDialer(handler model.Handler) *Dialer
```

This function is like `httpx.NewClient` and, specifically, it also
uses the current time as "zero" for subsequent events.

The `netx.Dialer` will also feature the following functions, to
be called before using the dialer:

```Go
func (c *Client) ConfigureDNS(network, address string) error
```

```Go
func (c *Client) SetCABundle(path string) error
```

```Go
func (c *Client) ForceSpecificSNI(sni string) error
```

`SetCABundle` and `ForceSpecificSNI` behave exactly like the same
methods of `httpx.Client`.

As far as `ConfigureDNS` is concerned it will work as follows:

* when `network` is `"system"`, the system resolver will be
used and no low-level events pertaining to the DNS will be
emitted to the configured `handler`. This will be the default.

* when `network` is `"udp"`, `address` must be a valid
string following the `"<ip_or_domain>(:<port>)*"` pattern. If
`<ip_or_domain>` is IPv6, it must be quoted using `[]`. If
`<port>` is omitted, we will use port `53`. This value will
indicate the code to use the selected DNS server using
UDP transport. We will be able to observe all events including
DNS messages sent and received.

* when `network` is `"tcp"`, everything will be like when
`network` is `"udp"`, except that we will speak the DNS
over TCP protocol with the configured server.

* when `network` is `"dot"`, `address` must be a valid
domain name, or IP address, of a DNS over TLS server to use. If
the port is omitted, we'll use port `853`. We will
observe all events, which of course include the results
of the TLS handshake with the server, the DNS messages
sent and received, etc.

* when `network` is `"doh"`, `address` must be a valid
URL of a DNS over HTTPS server to use. We will observe all
events, including the TLS handshake and HTTP events, the
DNS messages sent and received, etc.

Lastly, `netx.Dialer` will expose this API:

```Go
func (d *Dialer) NewResolver(network, address string) (model.DNSResolver, error)
```

The arguments have the same meaning of `ConfigureDNS` and
the will return an interface replacement for `net.Resolver`
as described below.
