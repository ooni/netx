// must be another package to avoid import cycle
package scoreboard_test

import (
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"reflect"
	"testing"

	"github.com/ooni/netx/modelx"
	"github.com/ooni/netx/x/scoreboard"
)

func TestIntegrationDNSBogon(t *testing.T) {
	board := &scoreboard.Board{}
	firstEntry := scoreboard.DNSBogonInfo{
		Addresses:              []string{"127.0.0.1"},
		DurationSinceBeginning: 1234,
		Domain:                 "www.x.org",
		FallbackPlan:           "antani",
	}
	board.AddDNSBogonInfo(firstEntry)
	secondEntry := scoreboard.DNSBogonInfo{
		Addresses:              []string{"10.0.0.1"},
		DurationSinceBeginning: 1235,
		Domain:                 "www.kernel.org",
		FallbackPlan:           "mascetti",
	}
	board.AddDNSBogonInfo(secondEntry)
	var result struct {
		DNSBogonINFO []scoreboard.DNSBogonInfo
	}
	data := board.Marshal()
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.DNSBogonINFO) != 2 {
		t.Fatal("invalid length")
	}
	if !reflect.DeepEqual(result.DNSBogonINFO[0], firstEntry) {
		t.Fatal("first entry mismatch")
	}
	if !reflect.DeepEqual(result.DNSBogonINFO[1], secondEntry) {
		t.Fatal("first entry mismatch")
	}
}

func TestIntegrationTLSHandshake(t *testing.T) {
	board := &scoreboard.Board{}
	board.MaybeTLSHandshakeReset(1, nil, nil)
	board.MaybeTLSHandshakeReset(4, nil, errors.New("antani"))
	board.MaybeTLSHandshakeReset(7, &url.URL{
		Host: "www.kernel.org",
	}, errors.New("connection_reset"))
	board.MaybeTLSHandshakeReset(11, &url.URL{
		Host: "www.x.org",
	}, &modelx.ErrWrapper{
		Failure: "connection_reset",
		WrappedErr: &net.OpError{
			Addr: &net.TCPAddr{
				IP:   net.IPv4(8, 8, 8, 8),
				Port: 53,
			},
		},
	})
	var result struct {
		TLSHandshakeReset []scoreboard.TLSHandshakeReset
	}
	data := board.Marshal()
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.TLSHandshakeReset) != 2 {
		t.Fatal("invalid length")
	}

	first := result.TLSHandshakeReset[0]
	if first.Address != "" {
		t.Fatal("unexpected address")
	}
	if first.Domain != "www.kernel.org" {
		t.Fatal("unexpected domain")
	}
	if first.DurationSinceBeginning != 7 {
		t.Fatal("unexpected duration")
	}
	expectedFollows := []string{"sni_blocking", "ip_valid_for_domain"}
	if !reflect.DeepEqual(first.RecommendedFollowups, expectedFollows) {
		t.Fatal("unexpected followups")
	}

	second := result.TLSHandshakeReset[1]
	if second.Address != "8.8.8.8:53" {
		t.Fatal("unexpected address")
	}
	if second.Domain != "www.x.org" {
		t.Fatal("unexpected domain")
	}
	if second.DurationSinceBeginning != 11 {
		t.Fatal("unexpected duration")
	}
	if !reflect.DeepEqual(second.RecommendedFollowups, expectedFollows) {
		t.Fatal("unexpected followups")
	}
}
