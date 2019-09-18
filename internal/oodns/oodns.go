// Package oodns is OONI's DNS client.
//
// This is currently experimental code that is not wired into the
// rest of the code. We want to understand if we can always use the
// github.com/miekg/dns client to implement dnsx.Client.
//
// If that is possible, then maybe we can fully replace the current
// situation in which we monkey patch Go's +netgo DNS client.
package oodns

import (
	"context"
	"errors"
	"net"

	"github.com/bassosimone/netx/dnsx"
	"github.com/bassosimone/netx/model"
	"github.com/miekg/dns"
)

// Client is OONI's DNS client. It is a simplistic client where we
// manually create and submit queries. It can use all the transports
// for DNS supported by this library, however.
type Client struct {
	handler   model.Handler
	transport dnsx.RoundTripper
}

// NewClient creates a new OONI DNS client instance.
func NewClient(handler model.Handler, t dnsx.RoundTripper) *Client {
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
	// TODO(bassosimone): wrap errors as net.DNSError
	// TODO(bassosimone): emit DNS messages
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
	// TODO(bassosimone): we are ignoring the context here
	var (
		querydata []byte
		replydata []byte
	)
	querydata, err = query.Pack()
	if err != nil {
		return
	}
	replydata, err = c.transport.RoundTrip(querydata)
	if err != nil {
		return
	}
	reply = new(dns.Msg)
	err = reply.Unpack(replydata)
	if err != nil {
		return
	}
	if reply.Rcode != dns.RcodeSuccess {
		err = errors.New("oodns: query failed")
		return
	}
	return
}
