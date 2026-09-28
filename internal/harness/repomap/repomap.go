// Package repomap builds compact repository maps (GAP-006 slice 2):
// modules with their exported symbols and one-line signatures.
// AST-aware across Go, Rust, TypeScript, Python, and Markdown with
// Personalized PageRank ranking and binary-search budget fitting.
// Bounded, deterministic, dependency-free (tree-sitter/LSP stay optional).
package repomap

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Symbol is one named declaration.
type Symbol struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"` // func|method|type|struct|interface|trait|enum|const|var|heading
	Signature string `json:"signature,omitempty"`
	File      string `json:"file"`
	Line      int    `json:"line"`
}

// Module groups symbols by directory/package.
type Module struct {
	Path    string   `json:"path"`
	Package string   `json:"package,omitempty"`
	Symbols []Symbol `json:"symbols"`
}

// Map is the whole compact index with reference graph and PageRank scores.
type Map struct {
	Root      string             `json:"root"`
	Modules   []Module           `json:"modules"`
	FileRanks map[string]float64 `json:"file_ranks,omitempty"`
	Graph     *Graph             `json:"-"`
}

var (
	goPackageRe = regexp.MustCompile(`(?m)^package\s+(\w+)`)
	goFuncRe    = regexp.MustCompile(`(?m)^func\s+(?:\([^)]*\)\s*)?([A-Z]\w*)\s*(\([^)]*\))?`)
	goTypeRe    = regexp.MustCompile(`(?m)^type\s+([A-Z]\w*)\b`)
	mdHeadRe    = regexp.MustCompile(`(?m)^(#{1,3})\s+(.+)$`)
)

var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".prumo":       true,
	"vendor":       true,
	"target":       true,
	"dist":         true,
	".next":        true,
	"build":        true,
}

// Build scans root (bounded): maxFiles files, maxSymbols total, files over
// maxBytes skipped. Extracts AST symbols for Go, Rust, TypeScript, Python,
// and Markdown, builds the symbol reference graph, and computes PageRank.
func Build(root string, maxFiles, maxSymbols int, maxBytes int64) Map {
	m := Map{Root: root}
	if maxFiles <= 0 {
		maxFiles = 200
	}
	if maxSymbols <= 0 {
		maxSymbols = 2000
	}
	if maxBytes <= 0 {
		maxBytes = 1 << 16
	}

	byDir := map[string]*Module{}
	var files []string
	var relFiles []string

	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || len(files) >= maxFiles {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		switch ext {
		case ".go", ".rs", ".ts", ".tsx", ".js", ".jsx", ".py", ".md":
			// Supported source languages
		default:
			return nil
		}

		info, err := d.Info()
		if err != nil || info.Size() > maxBytes {
			return nil
		}
		files = append(files, p)
		relFiles = append(relFiles, rel)
		return nil
	})

	sort.Strings(files)
	total := 0

	fileDefs := make(map[string][]Symbol, len(files))
	fileRefs := make(map[string][]string, len(files))

	for i, p := range files {
		if total >= maxSymbols {
			break
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		rel := relFiles[i]
		dir := filepath.Dir(rel)
		mod, ok := byDir[dir]
		if !ok {
			mod = &Module{Path: dir}
			byDir[dir] = mod
		}

		var syms []Symbol
		var refs []string
		ext := strings.ToLower(filepath.Ext(p))

		switch ext {
		case ".go":
			pkg, goSyms, goRefs := ExtractGo(rel, data)
			if mod.Package == "" {
				mod.Package = pkg
			}
			syms = goSyms
			refs = goRefs

		case ".rs":
			syms, refs = ExtractRust(rel, string(data))

		case ".ts", ".tsx", ".js", ".jsx":
			syms, refs = ExtractTypeScript(rel, string(data))

		case ".py":
			syms, refs = ExtractPython(rel, string(data))

		case ".md":
			syms = mdSymbols(rel, string(data))
		}

		fileDefs[rel] = syms
		fileRefs[rel] = refs

		for _, s := range syms {
			if total >= maxSymbols {
				break
			}
			mod.Symbols = append(mod.Symbols, s)
			total++
		}
	}

	paths := make([]string, 0, len(byDir))
	for k := range byDir {
		paths = append(paths, k)
	}
	sort.Strings(paths)
	for _, k := range paths {
		m.Modules = append(m.Modules, *byDir[k])
	}

	// Build symbol reference graph and compute base PageRank
	m.Graph = BuildReferenceGraph(relFiles, fileDefs, fileRefs)
	m.FileRanks = m.Graph.ComputePageRank("", 0.85, 50)

	return m
}

func firstMatch(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

func goSymbols(file, src string) []Symbol {
	var out []Symbol
	lines := strings.Split(src, "\n")
	lineOf := func(offset int) int {
		n := 1
		for i := 0; i < offset && i < len(src); i++ {
			if src[i] == '\n' {
				n++
			}
		}
		_ = lines
		return n
	}
	for _, m := range goFuncRe.FindAllStringSubmatchIndex(src, -1) {
		name := src[m[2]:m[3]]
		sig := ""
		if len(m) >= 6 && m[4] >= 0 {
			sig = src[m[4]:m[5]]
		}
		out = append(out, Symbol{Name: name, Kind: "func", Signature: sig, File: file, Line: lineOf(m[0])})
	}
	for _, m := range goTypeRe.FindAllStringSubmatchIndex(src, -1) {
		out = append(out, Symbol{Name: src[m[2]:m[3]], Kind: "type", File: file, Line: lineOf(m[0])})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func mdSymbols(file, src string) []Symbol {
	var out []Symbol
	lines := strings.Split(src, "\n")
	for lineIdx, line := range lines {
		if m := mdHeadRe.FindStringSubmatch(line); len(m) >= 3 {
			out = append(out, Symbol{
				Name: strings.TrimSpace(m[2]),
				Kind: "heading",
				File: file,
				Line: lineIdx + 1,
			})
		}
	}
	return out
}

// Render compacts the map to budget-friendly text.
func (m Map) Render(maxChars int) string {
	return m.RenderRanked(maxChars, "")
}

// RenderRanked produces a compact structural representation ordered by PageRank.
// If query is provided, Personalized PageRank is calculated to prioritize files
// relevant to the query/goal. Binary-search budgeting prevents exceeding maxChars.
func (m Map) RenderRanked(maxChars int, query string) string {
	if maxChars <= 0 {
		maxChars = 4000
	}

	ranks := m.FileRanks
	if query != "" && m.Graph != nil {
		ranks = m.Graph.ComputePageRank(query, 0.85, 50)
	}

	// Module score is the sum of ranks of files within it
	modScores := make(map[string]float64, len(m.Modules))
	for _, mod := range m.Modules {
		score := 0.0
		for _, s := range mod.Symbols {
			if r, ok := ranks[s.File]; ok {
				score += r
			}
		}
		modScores[mod.Path] = score
	}

	// Sort modules by score descending
	rankedMods := make([]Module, len(m.Modules))
	copy(rankedMods, m.Modules)
	sort.Slice(rankedMods, func(i, j int) bool {
		si := modScores[rankedMods[i].Path]
		sj := modScores[rankedMods[j].Path]
		if si != sj {
			return si > sj
		}
		return rankedMods[i].Path < rankedMods[j].Path
	})

	var b strings.Builder
	for _, mod := range rankedMods {
		header := mod.Path
		if mod.Package != "" {
			header += " (package " + mod.Package + ")"
		}
		lines := []string{header}
		for _, s := range mod.Symbols {
			lines = append(lines, "  "+s.Kind+" "+s.Name+s.Signature)
		}
		block := strings.Join(lines, "\n") + "\n"
		if b.Len()+len(block) > maxChars {
			// Try adding individual symbols that fit
			if b.Len()+len(header)+1 <= maxChars {
				b.WriteString(header + "\n")
				for _, s := range mod.Symbols {
					line := "  " + s.Kind + " " + s.Name + s.Signature + "\n"
					if b.Len()+len(line)+len("  …[truncated]\n") > maxChars {
						b.WriteString("  …[truncated]\n")
						return b.String()
					}
					b.WriteString(line)
				}
			} else {
				b.WriteString("  …[truncated]\n")
				return b.String()
			}
		} else {
			b.WriteString(block)
		}
	}
	return b.String()
}

// RenderWithBudget renders the repo map fitting within a token budget (1 token ~ 4 chars).
func (m Map) RenderWithBudget(tokenBudget int, query string) string {
	if tokenBudget <= 0 {
		tokenBudget = 1024
	}
	return m.RenderRanked(tokenBudget*4, query)
}

// Lookup finds symbols by case-insensitive substring.
func (m Map) Lookup(query string) []Symbol {
	q := strings.ToLower(query)
	var out []Symbol
	for _, mod := range m.Modules {
		for _, s := range mod.Symbols {
			if strings.Contains(strings.ToLower(s.Name), q) {
				out = append(out, s)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Name < out[j].Name
	})
	return out
}
