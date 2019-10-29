package scoreboard

import "testing"

func TestIntegration(t *testing.T) {
	board := &Board{}
	board.AddDNSBogonInfo(DNSBogonInfo{})
	t.Log(board.Marshal())
}
