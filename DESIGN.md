# OONI Network Extensions

| Author       | Simone Basso |
|--------------|--------------|
| Last-Updated | 2019-09-30   |
| Status       | open         |

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

A OONI experiment is expected to create instances of
our replacement objects, configure them properly,
then use our replacements, which are compatible with
standard library mechanisms to perform their task,
e.g. fetching a URL. After the measurement task
is completed, the experiment code will include the
low-level events into the measurement result object, and will walk
through the stream of events to determine in a more
precise way what could have gone wrong.

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
    Close                   *CloseEvent
    Connect                 *ConnectEvent
    DNSQuery                *DNSQueryEvent
    DNSReply                *DNSReplyEvent
    HTTPConnectionReady     *HTTPConnectionReadyEvent
    HTTPRequestStart        *HTTPRequestStartEvent
    HTTPRequestHeadersDone  *HTTPRequestHeadersDoneEvent
    HTTPRequestDone         *HTTPRequestDoneEvent
    HTTPResponseStart       *HTTPResponseStartEvent
    HTTPResponseHeadersDone *HTTPResponseHeadersDoneEvent
    HTTPResponseDone        *HTTPResponseDoneEvent
    Read                    *ReadEvent
    Resolve                 *ResolveEvent
    TLSHandshake            *TLSHandshakeEvent
    Write                   *WriteEvent
}
```

That is, it will contain a pointer for every event
that we support. The events processing code will
check what pointer or pointers are not `nil` to
known which event or events have occurred.

The following network-level events will be defined:

1. `CloseEvent`, indicating when a socket is closed
2. `ConnectEvent`, indicating the result of connecting
3. `Read`, indicating when a `read` completes
4. `Resolve`, indicating when a name resolution completes
5. `Write`, indicating when a `write` completes

The following DNS-level events will be defined:

1. `DNSQueryEvent`, containing the query data
2. `DNSReplyEvent`, containing the reply data

The following HTTP-level events will be defined:

1. `HTTPConnectionReadyEvent`, indicating when the connection
is ready to be used by HTTP code

2. `HTTPRequestStartEvent`, indicating when we start sending the request

3. `HTTPRequestHeadersDoneEvent`, indicating when we have sent the
request headers, and containing the sent headers

4. `HTTPRequestDoneEvent`, indicating when the whole request has been sent

5. `HTTPResponseStartEvent`, indicacting when we receive the first
byte of the HTTP response

6. `HTTPResponseHeadersDoneEvent`, indicating when we have received the
response headers, and containing headers and status code

7. `HTTPResponseDoneEvent`, indicating when we have received the
response body

Every event will include at the minimum this field:

```Go
    Time     time.Duration
```

This will be the time when the event occurred, relative to
a configured "zero" time. If an event pertains to a blocking
operation (i.e. `Read`), it will also contain this field:

```Go
    Duration time.Duration
```

This will be the amount time we have been waiting for
the event to occur. That is, in the case of `Read` the
amount of time we've been blocking waiting for the `Read`
operation to return a value or an error.

Every operation that can fail will have a field

```Go
    Error    error
```

This will indicate the error that occurred.

Measurement events will also contain contextual information
that is meaningful to the event itself. Since this is likely
to change as we improve our understanding of what could
be measured, as stated above, please see the current documentation
for more information on the structure of each event.

Every network event will be additionally identified by

```Go
    ConnID   int64
```

Where `ConnID` is the identifier of the connection and is
unique within a specific set of measurements.

Likewise, HTTP events will have their

```Go
    TransactionID int64
```

which will uniquely identify the round trip within a specific
set of measurements.

Because in this first PoC it has been deemed complex to access
the `ConnID` from the HTTP code, we have determined that we
will be using the five-tuple to join network and HTTP
events. Accordingly, both the `ConnectEvent` and
`HTTPConnectionReadyEvent` structures will thus include:

```Go
    LocalAddress  string
    Network       string
    RemoteAddress string
```

The problem of joining together network and HTTP level
measurements is currently not solved by this library. If
we perform a measurement at a time, however, this may
not be a big issue, because all the low-level events will
necessarily pertain to a single measurement, e.g., to
the fetching of a specific URL.

A subsequent revision of this specification will see
whether we can join measurements in a better way.

(As a contextual note, the problem of knowing the ID
of a connection is that we cannot wrap `*tls.Conn`
with a ConnID-aware-replacement that is compatible with `net.Conn`,
because that will confuse `net/http` and prevent using
`http2`. We could solve the problem to join automatically network
and lower-level events by implementing a goroutine
safe cache mapping the five tuple to a `ConnID`.)

### The github.com/ooni/netx/httpx package

This package will contain HTTP extensions. The core
structure that we will provide is as follows:

```Go
type Client struct {
  HTTPClient *http.Client
  Transport  Transport
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

* when `network` is `"netgo"`, we will try to use the DNS
resolver written in Go within the standard library (which is
know to work only on Unix), and we will use a bunch of
hacks to observe the events occurring during name resolutions,
such as `Read` and `Write` events. We will also be able to
record the DNS messages sent and received.

* when `network` is `"udp"`, `address` must be a valid
string following the `"<ip>:<port>"` pattern. This will
indicate the code to use the selected DNS server using
UDP transport. We will be able to observe all events including
DNS messages sent and received.

* when `network` is `"tcp"`, everything will be like when
`network` is `"udp"`, except that we will speak the DNS
over TCP protocol with the configured server.

* when `network` is `"dot"`, `address` must be a valid
domain name of a DNS over TLS server to use. We will
observe all events, which of course include the results
of the TLS handshake with the server, the DNS messages
sent and received, etc.

* when `network` is `"doh"`, `address` must be a valid
URL of a DNS over HTTPS server to use. We will observe all
events, including the TLS handshake and HTTP events, the
DNS messages sent and received, etc.

Lastly, `netx.Dialer` will expose this API:

```Go
func (d *Dialer) NewResolver(network, address string) (dnsx.Client, error)
```

The arguments have the same meaning of `ConfigureDNS` and
the will return an interface replacement for `net.Resolver`
as described below.

### The github.com/ooni/netx/dnsx package

This package will define an interface compatible with the
`net.Resolver` struct, such that its methods can be used
as replacaments for the golang stdlib `net.Resolver` methods:

```Go
type Client interface {
    LookupAddr(ctx context.Context, addr string) (names []string, err error)
    LookupCNAME(ctx context.Context, host string) (cname string, err error)
    LookupHost(ctx context.Context, hostname string) (addrs []string, err error)
    LookupMX(ctx context.Context, name string) ([]*net.MX, error)
    LookupNS(ctx context.Context, name string) ([]*net.NS, error)
}
```

## Future work

The current revision of this specification does not specify a
programmatic way of joining measurements occurring at different
levels (e.g. network and HTTP). This has been done under the
assumption that we will probably be able to understand the
sequence of events anyway, by looking at the timing, if we're
measuring a single URL at a time. We will implement and
deploy code conformant with this specification and see whether
this assumption is correct, or we need something else.
