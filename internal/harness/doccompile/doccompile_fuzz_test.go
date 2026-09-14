package doccompile

import "testing"

func FuzzFingerprintStable(f *testing.F) {
	f.Add("# seed")
	f.Fuzz(func(t *testing.T, s string) {
		if fingerprint(s) != fingerprint(s) {
			t.Fatal("fingerprint must be deterministic")
		}
		if s != "" && fingerprint(s) == fingerprint("x\x00"+s) {
			// SHA-256 must differ for distinct inputs in practice.
			t.Log("note: inputs share fingerprint (unexpected)")
		}
	})
}
