package tracing

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ooni/netx/internal/handlers/counthandler"
	"github.com/ooni/netx/internal/handlers/savinghandler"
	"github.com/ooni/netx/model"
)

func TestIntegrationWorks(t *testing.T) {
	const count = 3
	var wg sync.WaitGroup
	wg.Add(1)
	ctx := WithInfo(context.Background(), &Info{
		Handler: &counthandler.Handler{},
	})
	go func(ctx context.Context) {
		info := ContextInfo(ctx)
		for i := 0; i < count; i++ {
			time.Sleep(250 * time.Millisecond)
			info.Handler.OnMeasurement(model.Measurement{})
		}
		wg.Done()
	}(ctx)
	wg.Wait()
	if ContextInfo(ctx).Handler.(*counthandler.Handler).Count != 3 {
		t.Fatal("did not record all emitted measurements")
	}
}

func TestPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	WithInfo(context.Background(), nil)
}

func TestEmitTLSHandshakeStart(t *testing.T) {
	handler := &savinghandler.Handler{}
	info := &Info{Handler: handler}
	config := &tls.Config{
		NextProtos: []string{"h2"},
		ServerName: "antani",
	}
	info.EmitTLSHandshakeStart(config)
	if len(handler.All) != 1 {
		t.Fatal("no events have been saved")
	}
	if handler.All[0].TLSHandshakeStart == nil {
		t.Fatal("missing correct event")
	}
	evt := handler.All[0].TLSHandshakeStart
	if !reflect.DeepEqual(evt.Config.NextProtos, config.NextProtos) {
		t.Fatal("ALPN info not correctly saved")
	}
	if evt.Config.ServerName != config.ServerName {
		t.Fatal("SNI not correctly saved")
	}
}

func TestEmitTLSHandshakeDoneNoState(t *testing.T) {
	handler := &savinghandler.Handler{}
	info := &Info{Handler: handler}
	info.EmitTLSHandshakeDone(nil, errors.New("mocked error"))
	if len(handler.All) != 1 {
		t.Fatal("no events have been saved")
	}
	if handler.All[0].TLSHandshakeDone == nil {
		t.Fatal("missing correct event")
	}
	evt := handler.All[0].TLSHandshakeDone
	if evt.ConnectionState != nil {
		t.Fatal("unexpected ConnectionState value")
	}
}

func TestEmitTLSHandshakeDoneWithState(t *testing.T) {
	handler := &savinghandler.Handler{}
	info := &Info{Handler: handler}
	info.EmitTLSHandshakeDone(&tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{
			&x509.Certificate{
				Raw: []byte("0xdeadbeef"),
			},
		},
		Version: tls.VersionTLS10,
	}, errors.New("mocked error"))
	if len(handler.All) != 1 {
		t.Fatal("no events have been saved")
	}
	if handler.All[0].TLSHandshakeDone == nil {
		t.Fatal("missing correct event")
	}
	evt := handler.All[0].TLSHandshakeDone
	if evt.ConnectionState == nil {
		t.Fatal("unexpected ConnectionState value")
	}
	if evt.ConnectionState.Version != tls.VersionTLS10 {
		t.Fatal("unexpected TLS version")
	}
	if len(evt.ConnectionState.PeerCertificates) != 1 {
		t.Fatal("unexpected number of peer certificates")
	}
	cert := evt.ConnectionState.PeerCertificates[0]
	if !bytes.Equal(cert.Data, []byte("0xdeadbeef")) {
		t.Fatal("incorrectly saved certificate info")
	}
}
