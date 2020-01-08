// Package scoreboard contains the measurements scoreboard.
//
// This is an experimental package and may change/disappear
// at any time without any documentation.
package scoreboard

import (
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"sync"
	"time"
)

// Board contains what we learned during measurements.
type Board struct {
	DNSBogonInfo      []DNSBogonInfo
	TLSHandshakeReset []TLSHandshakeReset
	mu                sync.Mutex
}

// DNSBogonInfo contains info on bogon replies received when
// using the default resolver. To perform this measurement,
// you need to divert the resolver to nervousresolver.Resolver
// using MeasurementRoot.LookupHost.
type DNSBogonInfo struct {
	Addresses              []string
	DurationSinceBeginning time.Duration
	Domain                 string
	FallbackPlan           string
}

// TLSHandshakeReset contains info on a RST received when
// performing a TLS handshake with a server.
type TLSHandshakeReset struct {
	Address                string
	Domain                 string
	DurationSinceBeginning time.Duration
	RecommendedFollowups   []string
}

// AddDNSBogonInfo adds info on a DNS bogon reply
func (b *Board) AddDNSBogonInfo(info DNSBogonInfo) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.DNSBogonInfo = append(b.DNSBogonInfo, info)
}

// MaybeTLSHandshakeReset inspects the provided error and
// updates the scoreboard with TLS handshake RST info, when
// this kind of interference has been detected.
func (b *Board) MaybeTLSHandshakeReset(
	durationSinceBeginning time.Duration, URL *url.URL, err error,
) {
	// TODO(bassosimone): add also EOF here?
	if err == nil || err.Error() != "connection_reset" {
		return
	}
	var (
		opError    *net.OpError
		remoteAddr string
	)
	if errors.As(err, &opError) && opError.Addr != nil {
		remoteAddr = opError.Addr.String()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.TLSHandshakeReset = append(b.TLSHandshakeReset, TLSHandshakeReset{
		Address:                remoteAddr,
		Domain:                 URL.Hostname(),
		DurationSinceBeginning: durationSinceBeginning,
		RecommendedFollowups: []string{
			"sni_blocking",
			"ip_valid_for_domain",
		},
	})
}

// Marshal marshals the board in JSON format.
func (b *Board) Marshal() []byte {
	data, _ := json.Marshal(b)
	return data
}
