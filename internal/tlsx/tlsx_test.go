package tlsx_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"testing"

	"github.com/ooni/netx/internal/tlsx"
	"github.com/ooni/netx/model"
)

func TestExistent(t *testing.T) {
	pool, err := tlsx.ReadCABundle("../../testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
	if pool == nil {
		t.Fatal("expected non-nil pool here")
	}
}

func TestNonExistent(t *testing.T) {
	pool, err := tlsx.ReadCABundle("../../testdata/cacert-nonexistent.pem")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if pool != nil {
		t.Fatal("expected a nil pool here")
	}
}

type handler struct {
	t *testing.T
}

func (h *handler) OnMeasurement(m model.Measurement) {
	if m.TLSHandshake == nil {
		h.t.Fatal("missing required message")
	}
	if m.TLSHandshake.Config.ServerName != "antani" {
		h.t.Fatal("invalid server name")
	}
	if len(m.TLSHandshake.Config.NextProtos) != 2 {
		h.t.Fatal("unexpected NextProtos length")
	}
	if m.TLSHandshake.Config.NextProtos[0] != "foo" {
		h.t.Fatal("unexpected NextProtos[0] value")
	}
	if m.TLSHandshake.Config.NextProtos[1] != "bar" {
		h.t.Fatal("unexpected NextProtos[1] value")
	}
	if m.TLSHandshake.ConnectionState.CipherSuite != 17 {
		h.t.Fatal("invalid cipher suite")
	}
	if m.TLSHandshake.ConnectionState.NegotiatedProtocol != "xo" {
		h.t.Fatal("invalid negotiated protocol")
	}
	if !m.TLSHandshake.ConnectionState.NegotiatedProtocolIsMutual {
		h.t.Fatal("invalid negotiated protocol is mutuale")
	}
	if len(m.TLSHandshake.ConnectionState.PeerCertificates) != 2 {
		h.t.Fatal("invalid number of peer certificates")
	}
	if !bytes.Equal(
		m.TLSHandshake.ConnectionState.PeerCertificates[0].Data, []byte("abc"),
	) {
		h.t.Fatal("invalid value of first certificate")
	}
	if !bytes.Equal(
		m.TLSHandshake.ConnectionState.PeerCertificates[1].Data, []byte("def"),
	) {
		h.t.Fatal("invalid value of second certificate")
	}
	if m.TLSHandshake.ConnectionState.Version != 770 {
		h.t.Fatal("invalid version")
	}
	if m.TLSHandshake.Time != 500000000 {
		h.t.Fatal("invalid Time")
	}
	if m.TLSHandshake.Duration != 120000000 {
		h.t.Fatal("invalid Duration")
	}
	if m.TLSHandshake.Error.Error() != "mocked error" {
		h.t.Fatal("invalid error")
	}
}

func TestEmitTLSHandshakeEvent(t *testing.T) {
	tlsx.EmitTLSHandshakeEvent(
		&handler{t: t},
		tls.ConnectionState{
			CipherSuite:                17,
			NegotiatedProtocol:         "xo",
			NegotiatedProtocolIsMutual: true,
			PeerCertificates: []*x509.Certificate{
				&x509.Certificate{
					Raw: []byte("abc"),
				},
				&x509.Certificate{
					Raw: []byte("def"),
				},
			},
			Version: 770,
		},
		500000000,
		120000000,
		errors.New("mocked error"),
		&tls.Config{
			NextProtos: []string{"foo", "bar"},
			ServerName: "antani",
		},
	)
}
