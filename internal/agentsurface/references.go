package agentsurface

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ReferenceKind classifies a code reference carried by an agent surface.
type ReferenceKind string

const (
	RefPath    ReferenceKind = "path"
	RefDoc     ReferenceKind = "doc"
	RefCommand ReferenceKind = "command"
	RefTest    ReferenceKind = "test"
	RefSymbol  ReferenceKind = "symbol"
)

// Reference is one pointer extracted from instruction text.
type Reference struct {
	Kind  ReferenceKind `json:"kind"`
	Value string        `json:"value"`
}

// Finding is a stale or unresolvable reference (the context-rot signal).
type Finding struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
	Value  string `json:"value"`
	Detail string `json:"detail"`
}

var (
	codeSpan   = regexp.MustCompile("`([^`\\n]+)`")
	globChars  = regexp.MustCompile(`[*?[]`)
	urlPrefix  = regexp.MustCompile(`^[a-z][a-z0-9+.-]*://`)
	symbolName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*\(\)$`)
	testName   = regexp.MustCompile(`^Test[A-Za-z0-9_]+$`)
	knownTools = map[string]bool{
		"prumo": true, "go": true, "git": true, "npm": true, "pnpm": true,
		"yarn": true, "bun": true, "docker": true, "podman": true, "make": true,
	}
	pathExt = map[string]bool{
		".md": true, ".json": true, ".go": true, ".ts": true, ".tsx": true,
		".js": true, ".py": true, ".rs": true, ".yaml": true, ".yml": true,
		".toml": true, ".sh": true, ".sql": true,
	}
)

// ExtractReferences pulls the checkable code references out of instruction
// text. Only unambiguous shapes are classified: anything that looks like prose,
// a flag, a snippet or a placeholder is ignored so the guard stays precise.
func ExtractReferences(text string) []Reference {
	seen := map[string]bool{}
	var refs []Reference
	for _, match := range codeSpan.FindAllStringSubmatch(text, -1) {
		value := strings.TrimSpace(match[1])
		if !isCheckable(value) {
			continue
		}
		ref := classify(value)
		if ref == nil {
			continue
		}
		key := string(ref.Kind) + "\x00" + ref.Value
		if seen[key] {
			continue
		}
		seen[key] = true
		refs = append(refs, *ref)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Kind != refs[j].Kind {
			return refs[i].Kind < refs[j].Kind
		}
		return refs[i].Value < refs[j].Value
	})
	return refs
}

func isCheckable(value string) bool {
	if value == "" || len(value) > 200 {
		return false
	}
	if urlPrefix.MatchString(value) {
		return false
	}
	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "$") {
		return false
	}
	if strings.ContainsAny(value, "{}<>|=\n") {
		return false
	}
	if strings.Contains(value, "{{") || strings.Contains(value, "...") && !strings.HasPrefix(value, "go ") {
		return false
	}
	return true
}

func classify(value string) *Reference {
	fields := strings.Fields(value)
	switch {
	case len(fields) > 0 && knownTools[fields[0]] && (len(fields) > 1 || fields[0] == "go"):
		return &Reference{Kind: RefCommand, Value: value}
	case testName.MatchString(value):
		return &Reference{Kind: RefTest, Value: value}
	case symbolName.MatchString(value):
		return &Reference{Kind: RefSymbol, Value: value}
	}
	if strings.HasPrefix(value, "docs/") && (pathExt[strings.ToLower(filepath.Ext(value))] || globChars.MatchString(value)) {
		return &Reference{Kind: RefDoc, Value: value}
	}
	if strings.Contains(value, "/") && (pathExt[strings.ToLower(filepath.Ext(value))] || globChars.MatchString(value)) {
		return &Reference{Kind: RefPath, Value: value}
	}
	return nil
}

// Referencer is any artifact carrying checkable references.
type Referencer struct {
	Target string
	Text   string
}

// SourceIndex indexes repository Go sources so command and symbol references
// can be validated without shelling out.
//
// Two indexes are kept deliberately: symbols and test names legitimately live
// in _test.go files, but a command must be implemented in production code. If
// commands were validated against test files too, a guard could validate a
// reference merely because the test that checks it mentions the same word.
type SourceIndex struct {
	root       string
	byWord     map[string]bool
	production map[string]bool
}

// NewSourceIndex indexes Go sources under cmd/ and internal/.
func NewSourceIndex(root string) *SourceIndex {
	idx := &SourceIndex{root: root, byWord: map[string]bool{}, production: map[string]bool{}}
	for _, dir := range []string{"cmd", "internal"} {
		base := filepath.Join(root, dir)
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			words := wordPattern.FindAllString(string(data), -1)
			for _, word := range words {
				idx.byWord[word] = true
			}
			if !strings.HasSuffix(d.Name(), "_test.go") {
				for _, word := range words {
					idx.production[word] = true
				}
			}
			return nil
		})
	}
	return idx
}

var wordPattern = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// HasWord reports whether an identifier appears anywhere in the Go sources.
func (idx *SourceIndex) HasWord(word string) bool { return idx.byWord[word] }

// HasProductionWord reports whether an identifier appears in non-test sources.
func (idx *SourceIndex) HasProductionWord(word string) bool { return idx.production[word] }

// ValidateReferences resolves every reference against repository reality.
// Unresolvable references are findings, never silently ignored (W16.10).
func ValidateReferences(root string, referencers []Referencer) []Finding {
	index := NewSourceIndex(root)
	var findings []Finding
	for _, ref := range referencers {
		for _, r := range ExtractReferences(ref.Text) {
			if detail := validateReference(root, index, r); detail != "" {
				findings = append(findings, Finding{
					Kind: "stale-reference", Target: ref.Target, Value: r.Value,
					Detail: string(r.Kind) + " reference is no longer resolvable: " + detail,
				})
			}
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Target != findings[j].Target {
			return findings[i].Target < findings[j].Target
		}
		return findings[i].Value < findings[j].Value
	})
	return findings
}

func validateReference(root string, index *SourceIndex, r Reference) string {
	switch r.Kind {
	case RefPath, RefDoc:
		candidate := strings.TrimSuffix(r.Value, "/")
		if globChars.MatchString(candidate) {
			matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(candidate)))
			if err != nil || len(matches) == 0 {
				return "glob matches nothing"
			}
			return ""
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(candidate))); err != nil {
			return "path does not exist"
		}
		return ""
	case RefCommand:
		for _, token := range strings.Fields(r.Value)[1:] {
			if strings.HasPrefix(token, "-") || strings.ContainsAny(token, "./*$") {
				continue
			}
			if !index.HasProductionWord(token) {
				return "subcommand " + token + " is not implemented in cmd/ or internal/"
			}
		}
		return ""
	case RefTest:
		if !index.HasWord(r.Value) {
			return "test symbol not found in the repository"
		}
		return ""
	case RefSymbol:
		name := strings.TrimSuffix(r.Value, "()")
		name = name[strings.LastIndex(name, ".")+1:]
		if !index.HasWord(name) {
			return "symbol not found in the repository"
		}
		return ""
	}
	return ""
}
