// Package oodns is OONI's DNS client.
//
// This is currently experimental code that is not wired into the
// rest of the code. We want to understand if we can always use the
// github.com/miekg/dns client to implement model.DNSResolver.
//
// If that is possible, then maybe we can fully replace the current
// situation in which we monkey patch Go's +netgo DNS client.
package oodns

import (
	"context"
	"errors"
	"net"

	"github.com/miekg/dns"
	"github.com/ooni/netx/model"
)

// Client is OONI's DNS client. It is a simplistic client where we
// manually create and submit queries. It can use all the transports
// for DNS supported by this library, however.
type Client struct {
	handler   model.Handler
	transport model.DNSRoundTripper
}

// NewClient creates a new OONI DNS client instance.
func NewClient(handler model.Handler, t model.DNSRoundTripper) *Client {
	return &Client{
		handler:   handler,
		transport: t,
	}
}

var errNotImpl = errors.New("Not implemented")

// LookupAddr returns the name of the provided IP address
func (c *Client) LookupAddr(ctx context.Context, addr string) (names []string, err error) {
	err = errNotImpl
	return
}

// LookupCNAME returns the canonical name of a host
func (c *Client) LookupCNAME(ctx context.Context, host string) (cname string, err error) {
	err = errNotImpl
	return
}

// LookupHost returns the IP addresses of a host
func (c *Client) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	// TODO(ooni): wrap errors as net.DNSError
	// TODO(ooni): emit DNS messages
	var addrs []string
	var reply *dns.Msg
	reply, errA := c.roundTrip(ctx, c.newQueryWithQuestion(dns.Question{
		Name:   dns.Fqdn(hostname),
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}))
	if errA == nil {
		for _, answer := range reply.Answer {
			if rra, ok := answer.(*dns.A); ok {
				ip := rra.A
				addrs = append(addrs, ip.String())
			}
		}
	}
	reply, errAAAA := c.roundTrip(ctx, c.newQueryWithQuestion(dns.Question{
		Name:   dns.Fqdn(hostname),
		Qtype:  dns.TypeAAAA,
		Qclass: dns.ClassINET,
	}))
	if errAAAA == nil {
		for _, answer := range reply.Answer {
			if rra, ok := answer.(*dns.AAAA); ok {
				ip := rra.AAAA
				addrs = append(addrs, ip.String())
			}
		}
	}
	return LookupHostResult(addrs, errA, errAAAA)
}

// LookupHostResult computes the final result of LookupHost. You generally
// only care about this function when writing tests.
func LookupHostResult(addrs []string, errA, errAAAA error) ([]string, error) {
	if len(addrs) > 0 {
		return addrs, nil
	}
	if errA != nil {
		return nil, errA
	}
	if errAAAA != nil {
		return nil, errAAAA
	}
	return nil, errors.New("oodns: no response returned")
}

// LookupMX returns the MX records of a specific name
func (c *Client) LookupMX(ctx context.Context, name string) (mx []*net.MX, err error) {
	err = errNotImpl
	return
}

// LookupNS returns the NS records of a specific name
func (c *Client) LookupNS(ctx context.Context, name string) (ns []*net.NS, err error) {
	err = errNotImpl
	return
}

func (c *Client) newQueryWithQuestion(q dns.Question) (query *dns.Msg) {
	query = new(dns.Msg)
	query.Id = dns.Id()
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.Question[0] = q
	return
}

func (c *Client) roundTrip(ctx context.Context, query *dns.Msg) (reply *dns.Msg, err error) {
	return c.RoundTripEx(
		ctx, query, func(msg *dns.Msg) ([]byte, error) {
			return msg.Pack()
		},
		func(t model.DNSRoundTripper, query []byte) (reply []byte, err error) {
			// Pass ctx to round tripper for cancellation as well
			// as to propagate context information
			return t.RoundTrip(ctx, query)
		},
		func(msg *dns.Msg, data []byte) (err error) {
			return msg.Unpack(data)
		},
	)
}

// RoundTripEx is a mockable implementation of the piece
// of code that performs the DNS round trip.
func (c *Client) RoundTripEx(
	ctx context.Context,
	query *dns.Msg,
	pack func(msg *dns.Msg) ([]byte, error),
	roundTrip func(t model.DNSRoundTripper, query []byte) (reply []byte, err error),
	unpack func(msg *dns.Msg, data []byte) (err error),
) (reply *dns.Msg, err error) {
	// TODO(ooni): we are ignoring the context here
	var (
		querydata []byte
		replydata []byte
	)
	querydata, err = pack(query)
	if err != nil {
		return
	}
	replydata, err = roundTrip(c.transport, querydata)
	if err != nil {
		return
	}
	reply = new(dns.Msg)
	err = unpack(reply, replydata)
	if err != nil {
		return
	}
	if reply.Rcode != dns.RcodeSuccess {
		err = errors.New("oodns: query failed")
		return
	}
	return
}
