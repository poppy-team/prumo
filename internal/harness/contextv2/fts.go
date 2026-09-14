// Lexical retrieval (GAP-006 first slice): BM25-lite over workspace text
// files, fused as a score bonus into packing. Embeddings stay optional;
// LSP/symbol graph remain future slices. Bounded and deterministic.
package contextv2

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// FTSDoc is one scorable document.
type FTSDoc struct {
	Ref  string
	Text string
}

// Tokenize lowercases and splits on non-alphanumerics.
func Tokenize(s string) []string {
	lower := strings.ToLower(s)
	fields := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := fields[:0]
	for _, f := range fields {
		if len(f) > 1 {
			out = append(out, f)
		}
	}
	return out
}

// BM25 scores docs against terms; returns scores sorted desc. k1/b follow
// textbook defaults; corpus beyond maxDocs is truncated deterministically.
func BM25(corpus []FTSDoc, terms []string, maxDocs int) []Item {
	if len(corpus) == 0 || len(terms) == 0 {
		return nil
	}
	docs := corpus
	if len(docs) > maxDocs {
		docs = docs[:maxDocs]
	}
	N := float64(len(docs))
	tf := make([]map[string]int, len(docs))
	lengths := make([]int, len(docs))
	var totalLen int
	for i, d := range docs {
		tf[i] = map[string]int{}
		for _, tok := range Tokenize(d.Text) {
			tf[i][tok]++
			lengths[i]++
		}
		totalLen += lengths[i]
	}
	avgLen := float64(totalLen) / float64(len(docs))
	if avgLen == 0 {
		return nil
	}
	df := map[string]int{}
	for _, m := range tf {
		for tok := range m {
			df[tok]++
		}
	}
	const k1, b = 1.2, 0.75
	type scored struct {
		ref   string
		score float64
	}
	var out []scored
	for i, d := range docs {
		var s float64
		for _, term := range terms {
			t := strings.ToLower(term)
			f := float64(tf[i][t])
			if f == 0 {
				continue
			}
			idf := math.Log(1 + (N-float64(df[t])+0.5)/(float64(df[t])+0.5))
			den := f + k1*(1-b+b*float64(lengths[i])/avgLen)
			s += idf * (f * (k1 + 1)) / den
		}
		if s > 0 {
			out = append(out, scored{d.Ref, s})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].ref < out[j].ref
	})
	items := make([]Item, 0, len(out))
	for _, s := range out {
		items = append(items, Item{Ref: s.ref, Score: s.score, Method: "fts"})
	}
	return items
}

// ftsBoost reads candidate files (bounded) and returns ref→bonus.
func ftsBoost(root string, refs []string, terms []string) map[string]float64 {
	corpus := []FTSDoc{}
	for _, ref := range refs {
		rel := strings.TrimPrefix(ref, "file:")
		if rel == ref {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil || len(data) > 1<<14 || !isTextBytes(data) {
			continue
		}
		corpus = append(corpus, FTSDoc{Ref: ref, Text: string(data)})
		if len(corpus) >= 100 {
			break
		}
	}
	sort.Slice(corpus, func(i, j int) bool { return corpus[i].Ref < corpus[j].Ref })
	ranked := BM25(corpus, terms, 100)
	bonus := map[string]float64{}
	var top float64
	for _, it := range ranked {
		if it.Score > top {
			top = it.Score
		}
	}
	if top == 0 {
		return bonus
	}
	for _, it := range ranked {
		// Normalized bonus up to +0.3 for the best lexical match.
		bonus[it.Ref] = 0.3 * it.Score / top
	}
	return bonus
}

func isTextBytes(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return false
		}
	}
	return true
}
