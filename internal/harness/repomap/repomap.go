// Package repomap builds compact repository maps (GAP-006 slice 2):
// modules with their exported-feeling symbols and one-line signatures.
// Go-aware via lightweight scanning (package clause + func/type decls);
// other text files contribute outline entries (markdown headings).
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
	Kind      string `json:"kind"` // func|type|const|var|method|heading
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

// Map is the whole compact index.
type Map struct {
	Root    string   `json:"root"`
	Modules []Module `json:"modules"`
}

var (
	goPackageRe = regexp.MustCompile(`(?m)^package\s+(\w+)`)
	goFuncRe    = regexp.MustCompile(`(?m)^func\s+(?:\([^)]*\)\s*)?([A-Z]\w*)\s*(\([^)]*\))?`)
	goTypeRe    = regexp.MustCompile(`(?m)^type\s+([A-Z]\w*)\b`)
	mdHeadRe    = regexp.MustCompile(`(?m)^(#{1,3})\s+(.+)$`)
)

var skipDirs = map[string]bool{".git": true, "node_modules": true, ".prumo": true, "vendor": true, "target": true, "dist": true}

// Build scans root (bounded): maxFiles files, maxSymbols total, files over
// maxBytes skipped. Only exported Go identifiers + md headings are indexed.
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
		if ext != ".go" && ext != ".md" {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxBytes {
			return nil
		}
		files = append(files, p)
		return nil
	})
	sort.Strings(files)
	total := 0
	for _, p := range files {
		if total >= maxSymbols {
			break
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(root, p)
		dir := filepath.Dir(rel)
		mod, ok := byDir[dir]
		if !ok {
			mod = &Module{Path: dir}
			byDir[dir] = mod
		}
		var syms []Symbol
		if strings.HasSuffix(p, ".go") {
			mod.Package = firstMatch(goPackageRe, string(data))
			syms = goSymbols(rel, string(data))
		} else {
			syms = mdSymbols(rel, string(data))
		}
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
	for i, m := range mdHeadRe.FindAllStringSubmatch(src, -1) {
		_ = i
		out = append(out, Symbol{Name: strings.TrimSpace(m[2]), Kind: "heading", File: file})
	}
	return out
}

// Render compacts the map to budget-friendly text.
func (m Map) Render(maxChars int) string {
	if maxChars <= 0 {
		maxChars = 4000
	}
	var b strings.Builder
	for _, mod := range m.Modules {
		header := mod.Path
		if mod.Package != "" {
			header += " (package " + mod.Package + ")"
		}
		b.WriteString(header + "\n")
		for _, s := range mod.Symbols {
			line := "  " + s.Kind + " " + s.Name + s.Signature + "\n"
			if b.Len()+len(line) > maxChars {
				b.WriteString("  …[truncated]\n")
				return b.String()
			}
			b.WriteString(line)
		}
	}
	return b.String()
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
