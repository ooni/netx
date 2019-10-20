// Package dnsx contains OONI's DNS extensions.
//
// Because this package was often causing import loop headaches we
// have moved everything inside of model. Specifically, adding to
// this file functions such as NewResolver or NewTransport will be
// conducive to such loops. As such, this file is unusable unless
// we move the definition, say, to model, and we just keep here the
// names originally defined by the design document.
package dnsx

import "github.com/ooni/netx/model"

// Client is a DNS client. The *net.Resolver used by Go implements
// this interface, but other implementations are possible.
type Client model.DNSClient

// RoundTripper represents an abstract DNS transport. This name
// was not originally specified inside of the design document but
// we've had it here for a while and may be useful later.
type RoundTripper model.DNSRoundTripper
