package repomap

import (
	"math"
	"sort"
	"strings"
)

// Graph represents the directed symbol dependency graph between files.
type Graph struct {
	Files     []string
	OutEdges  map[string]map[string]int
	InEdges   map[string]map[string]int
	OutDegree map[string]int
}

// NewGraph initializes an empty dependency graph.
func NewGraph(files []string) *Graph {
	g := &Graph{
		Files:     files,
		OutEdges:  make(map[string]map[string]int, len(files)),
		InEdges:   make(map[string]map[string]int, len(files)),
		OutDegree: make(map[string]int, len(files)),
	}
	for _, f := range files {
		g.OutEdges[f] = make(map[string]int)
		g.InEdges[f] = make(map[string]int)
	}
	return g
}

// AddEdge records a reference from src to dst with count weight.
func (g *Graph) AddEdge(src, dst string, weight int) {
	if src == "" || dst == "" || src == dst {
		return
	}
	if weight <= 0 {
		weight = 1
	}
	if g.OutEdges[src] == nil {
		g.OutEdges[src] = make(map[string]int)
	}
	if g.InEdges[dst] == nil {
		g.InEdges[dst] = make(map[string]int)
	}
	g.OutEdges[src][dst] += weight
	g.InEdges[dst][src] += weight
	g.OutDegree[src] += weight
}

// BuildReferenceGraph constructs edges between files based on symbol definitions and usages.
func BuildReferenceGraph(files []string, fileDefs map[string][]Symbol, fileRefs map[string][]string) *Graph {
	g := NewGraph(files)

	// Map symbol name to defining files
	symToFiles := make(map[string][]string)
	for file, syms := range fileDefs {
		for _, s := range syms {
			if s.Name != "" {
				symToFiles[s.Name] = append(symToFiles[s.Name], file)
			}
		}
	}

	// For each file's references, add edges to defining files
	for srcFile, refs := range fileRefs {
		refCounts := make(map[string]int)
		for _, ref := range refs {
			if targetFiles, ok := symToFiles[ref]; ok {
				for _, targetFile := range targetFiles {
					if targetFile != srcFile {
						refCounts[targetFile]++
					}
				}
			}
		}
		for targetFile, count := range refCounts {
			g.AddEdge(srcFile, targetFile, count)
		}
	}

	return g
}

// ComputePageRank calculates Personalized PageRank over the reference graph.
// If query is provided, seeds are files that match the query tokens (Personalized PageRank).
// If no seeds match, uniform restart is used.
func (g *Graph) ComputePageRank(query string, damping float64, maxIter int) map[string]float64 {
	n := len(g.Files)
	if n == 0 {
		return make(map[string]float64)
	}
	if damping <= 0 || damping >= 1.0 {
		damping = 0.85
	}
	if maxIter <= 0 {
		maxIter = 50
	}

	// 1. Identify seed files for personalization
	seeds := g.findSeeds(query)
	personalization := make(map[string]float64, n)
	if len(seeds) > 0 {
		prob := 1.0 / float64(len(seeds))
		for _, s := range seeds {
			personalization[s] = prob
		}
	} else {
		prob := 1.0 / float64(n)
		for _, f := range g.Files {
			personalization[f] = prob
		}
	}

	// 2. Initialize rank vector
	ranks := make(map[string]float64, n)
	for _, f := range g.Files {
		ranks[f] = personalization[f]
	}

	// 3. Power iteration
	for iter := 0; iter < maxIter; iter++ {
		nextRanks := make(map[string]float64, n)

		// Sum rank of dangling nodes (out-degree 0)
		danglingSum := 0.0
		for _, f := range g.Files {
			if g.OutDegree[f] == 0 {
				danglingSum += ranks[f]
			}
		}

		// Calculate new rank for each node
		maxDiff := 0.0
		for _, dst := range g.Files {
			inSum := 0.0
			for src, weight := range g.InEdges[dst] {
				outDeg := g.OutDegree[src]
				if outDeg > 0 {
					inSum += ranks[src] * (float64(weight) / float64(outDeg))
				}
			}

			// P(dst) = (1 - d) * v[dst] + d * (danglingSum * v[dst] + inSum)
			v := personalization[dst]
			newRank := (1.0-damping)*v + damping*(danglingSum*v+inSum)
			nextRanks[dst] = newRank

			diff := math.Abs(newRank - ranks[dst])
			if diff > maxDiff {
				maxDiff = diff
			}
		}

		ranks = nextRanks
		if maxDiff < 1e-6 {
			break
		}
	}

	// 4. Normalize sum to 1.0
	total := 0.0
	for _, r := range ranks {
		total += r
	}
	if total > 0 {
		for f := range ranks {
			ranks[f] /= total
		}
	}

	return ranks
}

// findSeeds locates files whose paths or basenames contain words from the query.
func (g *Graph) findSeeds(query string) []string {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return nil
	}
	words := strings.Fields(q)
	var seeds []string
	for _, f := range g.Files {
		fLower := strings.ToLower(f)
		for _, w := range words {
			if len(w) >= 3 && strings.Contains(fLower, w) {
				seeds = append(seeds, f)
				break
			}
		}
	}
	return seeds
}

// RankedItem associates a file path or module with its PageRank score.
type RankedItem struct {
	Path  string
	Score float64
}

// SortRanked sorts paths by score descending.
func SortRanked(scores map[string]float64) []RankedItem {
	items := make([]RankedItem, 0, len(scores))
	for path, score := range scores {
		items = append(items, RankedItem{Path: path, Score: score})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		return items[i].Path < items[j].Path
	})
	return items
}
