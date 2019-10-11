package tlsx_test

import (
	"testing"

	"github.com/ooni/netx/internal/tlsx"
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
