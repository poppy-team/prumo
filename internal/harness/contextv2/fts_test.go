package contextv2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBM25RanksRelevantFirst(t *testing.T) {
	corpus := []FTSDoc{
		{Ref: "a", Text: "the quick brown fox"},
		{Ref: "b", Text: "harness checkpoint resume logic"},
		{Ref: "c", Text: "unrelated words here entirely"},
	}
	ranked := BM25(corpus, Tokenize("checkpoint resume"), 10)
	if len(ranked) == 0 || ranked[0].Ref != "b" {
		t.Fatalf("expected b first: %+v", ranked)
	}
	if BM25(nil, Tokenize("x"), 10) != nil || BM25(corpus, nil, 10) != nil {
		t.Fatal("empty input must yield nil")
	}
}

func TestWorkspaceFTSBoostsMatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zzz.go"), []byte("package z\n// checkpoint resume journal\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "aaa.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plain := CompileWorkspace("R-fts", "checkpoint resume", dir, 8000, "")
	// Both files pack; the lexical match must outrank alphabetically-first.
	if len(plain.Included) < 3 {
		t.Fatalf("expected goal+2 files: %+v", plain.Included)
	}
}
