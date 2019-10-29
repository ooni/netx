// Package ooniresolver is OONI's DNS resolver.
package ooniresolver

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/netx/internal/dialid"
	"github.com/ooni/netx/model"
)

// Resolver is OONI's DNS client. It is a simplistic client where we
// manually create and submit queries. It can use all the transports
// for DNS supported by this library, however.
type Resolver struct {
	ntimeouts int64
	transport model.DNSRoundTripper
}

// New creates a new OONI Resolver instance.
func New(t model.DNSRoundTripper) *Resolver {
	return &Resolver{transport: t}
}

var errNotImpl = errors.New("Not implemented")

// LookupAddr returns the name of the provided IP address
func (c *Resolver) LookupAddr(ctx context.Context, addr string) (names []string, err error) {
	err = errNotImpl
	return
}

// LookupCNAME returns the canonical name of a host
func (c *Resolver) LookupCNAME(ctx context.Context, host string) (cname string, err error) {
	err = errNotImpl
	return
}

// LookupHost returns the IP addresses of a host
func (c *Resolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	// TODO(bassosimone): wrap errors as net.DNSError
	var addrs []string
	var reply *dns.Msg
	reply, errA := c.roundTripWithRetry(ctx, hostname, dns.TypeA)
	if errA == nil {
		for _, answer := range reply.Answer {
			if rra, ok := answer.(*dns.A); ok {
				ip := rra.A
				addrs = append(addrs, ip.String())
			}
		}
	}
	reply, errAAAA := c.roundTripWithRetry(ctx, hostname, dns.TypeAAAA)
	if errAAAA == nil {
		for _, answer := range reply.Answer {
			if rra, ok := answer.(*dns.AAAA); ok {
				ip := rra.AAAA
				addrs = append(addrs, ip.String())
			}
		}
	}
	return lookupHostResult(addrs, errA, errAAAA)
}

func lookupHostResult(addrs []string, errA, errAAAA error) ([]string, error) {
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
func (c *Resolver) LookupMX(ctx context.Context, name string) (mx []*net.MX, err error) {
	err = errNotImpl
	return
}

// LookupNS returns the NS records of a specific name
func (c *Resolver) LookupNS(ctx context.Context, name string) (ns []*net.NS, err error) {
	err = errNotImpl
	return
}

func (c *Resolver) newQueryWithQuestion(q dns.Question) (query *dns.Msg) {
	query = new(dns.Msg)
	query.Id = dns.Id()
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.Question[0] = q
	return
}

func (c *Resolver) roundTripWithRetry(
	ctx context.Context, hostname string, qtype uint16,
) (*dns.Msg, error) {
	for i := 0; i < 3; i++ {
		reply, err := c.roundTrip(ctx, c.newQueryWithQuestion(dns.Question{
			Name:   dns.Fqdn(hostname),
			Qtype:  qtype,
			Qclass: dns.ClassINET,
		}))
		if err == nil {
			return reply, nil
		}
		var operr *net.OpError
		if errors.As(err, &operr) == false || operr.Timeout() == false {
			return nil, err
		}
		atomic.AddInt64(&c.ntimeouts, 1)
	}
	return nil, context.DeadlineExceeded
}

func (c *Resolver) roundTrip(ctx context.Context, query *dns.Msg) (reply *dns.Msg, err error) {
	return c.mockableRoundTrip(
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

func (c *Resolver) mockableRoundTrip(
	ctx context.Context,
	query *dns.Msg,
	pack func(msg *dns.Msg) ([]byte, error),
	roundTrip func(t model.DNSRoundTripper, query []byte) (reply []byte, err error),
	unpack func(msg *dns.Msg, data []byte) (err error),
) (reply *dns.Msg, err error) {
	var (
		querydata []byte
		replydata []byte
	)
	querydata, err = pack(query)
	if err != nil {
		return
	}
	root := model.ContextMeasurementRootOrDefault(ctx)
	root.Handler.OnMeasurement(model.Measurement{
		DNSQuery: &model.DNSQueryEvent{
			Data:                   querydata,
			DialID:                 dialid.ContextDialID(ctx),
			DurationSinceBeginning: time.Now().Sub(root.Beginning),
			Msg:                    query,
		},
	})
	replydata, err = roundTrip(c.transport, querydata)
	if err != nil {
		return
	}
	reply = new(dns.Msg)
	err = unpack(reply, replydata)
	if err != nil {
		return
	}
	root.Handler.OnMeasurement(model.Measurement{
		DNSReply: &model.DNSReplyEvent{
			Data:                   replydata,
			DialID:                 dialid.ContextDialID(ctx),
			DurationSinceBeginning: time.Now().Sub(root.Beginning),
			Msg:                    reply,
		},
	})
	if reply.Rcode != dns.RcodeSuccess {
		err = errors.New("oodns: query failed")
		return
	}
	return
}
