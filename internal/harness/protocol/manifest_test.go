package protocol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found walking up")
		}
		dir = parent
	}
}

// TestManifestMatchesCheckedInFile keeps schemas/protocol-manifest.json and
// Manifest() in sync: the file is the published IDL, the code is the truth.
func TestManifestMatchesCheckedInFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "schemas", "protocol-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		XManifest map[string]any `json:"x-manifest"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(Manifest())
	if err != nil {
		t.Fatal(err)
	}
	var gotDecoded map[string]any
	if err := json.Unmarshal(got, &gotDecoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotDecoded, doc.XManifest) {
		want, _ := json.MarshalIndent(doc.XManifest, "", "  ")
		t.Fatalf("manifest drift: code Manifest() != schemas/protocol-manifest.json x-manifest.\nfile:\n%s\ngot:\n%s", want, got)
	}
}

func TestOpsHaveArgs(t *testing.T) {
	for _, op := range Ops {
		if _, ok := OpArgs[op]; !ok {
			t.Fatalf("op %q missing args doc", op)
		}
	}
}
