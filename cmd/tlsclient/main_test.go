package main

import (
	"testing"

	"github.com/ooni/netx/cmd/common"
)

func TestIntegration(t *testing.T) {
	main()
}

func TestHelp(t *testing.T) {
	*common.FlagHelp = true
	main()
	*common.FlagHelp = false
}
