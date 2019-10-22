// Package tlsconf helps with configuring TLS
package tlsconf

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
)

// SetCABundle configures conf to use a specific CA bundle.
func SetCABundle(conf *tls.Config, path string) error {
	cert, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(cert)
	conf.RootCAs = pool
	return nil
}

// ForceSpecificSNI sets a specifici SNI in conf.
func ForceSpecificSNI(conf *tls.Config, sni string) error {
	conf.ServerName = sni
	return nil
}
