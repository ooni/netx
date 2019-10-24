// Package internal contains internal code
package internal

import (
	"crypto/sha256"
	"fmt"
	"net"
)

// ConnHash computes the connection hash
func ConnHash(conn net.Conn) string {
	local := conn.LocalAddr()
	remote := conn.RemoteAddr()
	network := local.Network()
	slug := network + local.String() + remote.String()
	sum := sha256.Sum256([]byte(slug))
	return fmt.Sprintf("%x", sum)
}
