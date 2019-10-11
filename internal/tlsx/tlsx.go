// Package tlsx contains crypto/tls extensions
package tlsx

import (
	"crypto/x509"
	"io/ioutil"
)

// ReadCABundle read a CA bundle from file
func ReadCABundle(path string) (*x509.CertPool, error) {
	cert, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(cert)
	return pool, nil
}
