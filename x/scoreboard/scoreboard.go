// Package scoreboard contains the measurements scoreboard.
//
// This is currently experimental code.
package scoreboard

import (
	"encoding/json"
	"sync"
	"time"
)

// Board contains what we learned during measurements.
type Board struct {
	DNSBogonInfo []DNSBogonInfo
	mu           sync.Mutex
}

// DNSBogonInfo contains info on bogon replies received when
// using the default resolver. To perform this measurement,
// you need to divert the resolver to nervousresolver.Resolver
// using MeasurementRoot.LookupHost.
type DNSBogonInfo struct {
	Addresses              []string
	DurationSinceBeginning time.Duration
	FollowupAction         string
	Hostname               string
}

// AddDNSBogonInfo adds info on a DNS bogon reply
func (b *Board) AddDNSBogonInfo(info DNSBogonInfo) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.DNSBogonInfo = append(b.DNSBogonInfo, info)
}

// Marshal marshals the board in JSON format.
func (b *Board) Marshal() string {
	data, _ := json.Marshal(b)
	return string(data)
}
