// must be another package to avoid import cycle
package scoreboard_test

import (
	"errors"
	"net"
	"net/url"
	"testing"

	"github.com/ooni/netx/model"
	"github.com/ooni/netx/x/scoreboard"
)

func TestIntegrationDNSBogon(t *testing.T) {
	board := &scoreboard.Board{}
	board.AddDNSBogonInfo(scoreboard.DNSBogonInfo{})
	t.Log(board.Marshal())
}

func TestIntegrationTLSHandshakeResetNil(t *testing.T) {
	board := &scoreboard.Board{}
	board.MaybeTLSHandshakeReset(0, nil, nil)
	t.Log(board.Marshal())
}

func TestIntegrationTLSHandshakeResetNotReset(t *testing.T) {
	board := &scoreboard.Board{}
	board.MaybeTLSHandshakeReset(0, nil, errors.New("antani"))
	t.Log(board.Marshal())
}

func TestIntegrationTLSHandshakeResetIsReset(t *testing.T) {
	board := &scoreboard.Board{}
	board.MaybeTLSHandshakeReset(0, &url.URL{}, errors.New("connection_reset"))
	t.Log(board.Marshal())
}

func TestIntegrationTLSHandshakeResetIsResetWithAddr(t *testing.T) {
	board := &scoreboard.Board{}
	board.MaybeTLSHandshakeReset(
		0, &url.URL{}, &model.ErrWrapper{
			Failure: "connection_reset",
			WrappedErr: &net.OpError{
				Addr: &net.TCPAddr{
					IP:   net.IPv4(8, 8, 8, 8),
					Port: 53,
				},
			},
		},
	)
	t.Log(board.Marshal())
}
