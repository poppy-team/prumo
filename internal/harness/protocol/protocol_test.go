package protocol

import "testing"

func TestNegotiate(t *testing.T) {
	if _, ok, _ := Negotiate(Version); !ok {
		t.Fatal("same version must be compatible")
	}
	if _, ok, _ := Negotiate("99.0.0"); ok {
		t.Fatal("major mismatch must be incompatible")
	}
	if _, _, err := Negotiate(""); err == nil {
		t.Fatal("empty version must error")
	}
	if _, _, err := Negotiate("bogus"); err == nil {
		t.Fatal("malformed version must error")
	}
	if len(Schemas) == 0 {
		t.Fatal("protocol must ship schemas")
	}
}
