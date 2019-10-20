// Package connector contains the generic connector interface
package connector

import (
	"context"
	"net"
)

// Model is the model of any abstract connector
type Model interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}
