package contextv2

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkCompileWorkspace(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < 50; i++ {
		content := fmt.Sprintf("package p%d\n// checkpoint resume budget tokens\nfunc f%d() {}\n", i, i)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.go", i)), []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CompileWorkspace("R-bench", "checkpoint resume budget", dir, 8000, "L1")
	}
}

func BenchmarkBM25(b *testing.B) {
	corpus := make([]FTSDoc, 100)
	for i := range corpus {
		corpus[i] = FTSDoc{Ref: fmt.Sprintf("f%d", i), Text: "harness checkpoint resume budget tokens model provider gateway"}
	}
	terms := Tokenize("checkpoint resume budget")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BM25(corpus, terms, 100)
	}
}
