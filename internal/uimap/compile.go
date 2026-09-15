package uimap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Options adjust one compilation.
type Options struct {
	// Write emits the projections.
	Write bool
	// Targets narrows the projections for this run, whatever the project
	// configuration says. Narrowing is allowed; widening is not, so a command
	// line cannot produce an artifact the project decided against.
	Targets []Target
	// Path overrides the canonical map location.
	Path string
	// Deriver replaces the derivation adapter. Nil uses the Go deriver.
	Deriver SymbolDeriver
}

// Result is everything one compilation produced.
type Result struct {
	Config      Config
	Map         Map
	Derived     Derived
	Report      Report
	Projections []Projection
	// Skipped is set when the project does not want an interface map. The
	// reason is carried so a report can state it instead of printing nothing.
	Skipped bool
	Reason  string
}

// Compile runs the whole pipeline for a project root.
//
// The order matters and is the point of the pipeline: validate the declaration
// before deriving, derive before resolving symbols, and merge only after both
// exist. Merging first would let a derived element participate in validation as
// if it had been declared, which is exactly the authority inversion the
// declared-wins rule exists to prevent.
func Compile(root string, opts Options) (Result, error) {
	cfg, err := ResolveConfig(root)
	if err != nil {
		return Result{}, err
	}
	if opts.Path != "" {
		cfg.Path = opts.Path
	}
	if len(opts.Targets) > 0 {
		narrowed := make([]Target, 0, len(opts.Targets))
		for _, target := range opts.Targets {
			if cfg.Wants(target) {
				narrowed = append(narrowed, target)
			}
		}
		if len(narrowed) == 0 {
			return Result{}, fmt.Errorf("uimap: none of %v is enabled for this project (enabled: %v)",
				opts.Targets, cfg.Targets)
		}
		cfg.Targets = orderTargets(narrowed)
	}
	if !cfg.Enabled {
		return Result{Config: cfg, Skipped: true, Reason: cfg.Reason}, nil
	}

	m, err := Load(cfg.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{}, fmt.Errorf("uimap: an interface map is expected at %s but does not exist; "+
				"create it, or disable the map with `ui.interface_map.enabled: false`", cfg.Path)
		}
		return Result{}, err
	}
	vocab, err := LoadVocabulary(root, m)
	if err != nil {
		return Result{}, err
	}

	deriver := opts.Deriver
	if deriver == nil {
		deriver = GoSymbolDeriver{}
	}
	derived, err := Derive(root, m, deriver)
	if err != nil {
		return Result{}, err
	}

	merged := Merge(m, derived, cfg.Derivation)
	resolver := NewSymbolResolver(derived)

	report := Validate(root, merged.Map, vocab, resolver)
	report.Derived = len(derived.Symbols)
	report.Undeclared = len(merged.Undeclared)
	if len(derived.Sources) == 0 && claimsImplementation(m) {
		report.Findings = append(report.Findings, Finding{
			Code: "missing-derivation-scope", Severity: SeverityError,
			Message: "map claims implemented elements but declares no derivation sources",
			Hint:    "declare `derivation.sources`, e.g. [\"tui/**/*.go\"], so the symbols can be proven to exist",
		})
	}
	if cfg.Derivation == "verify-only" {
		for _, symbol := range merged.Undeclared {
			report.Findings = append(report.Findings, Finding{
				Code: "undeclared-element", Severity: SeverityWarning,
				Message: fmt.Sprintf("implementation declares %q, which the map does not describe", symbol),
				Hint:    "describe it, or set `ui.interface_map.derivation: fill-gaps` to list it for review",
			})
		}
	}

	result := Result{Config: cfg, Map: merged.Map, Derived: derived, Report: report}
	result.Projections = Project(merged.Map, cfg, report, derived)
	if opts.Write {
		if err := WriteProjections(root, result.Projections); err != nil {
			return Result{}, err
		}
	}
	return result, nil
}

// claimsImplementation reports whether any node asserts a symbol.
func claimsImplementation(m Map) bool {
	for _, walk := range m.Flatten() {
		impl := walk.Node.Implementation
		if impl == nil {
			continue
		}
		if impl.Status != "not-implemented" && impl.Symbol != "" {
			return true
		}
	}
	return false
}

// ReportText renders the human report.
func ReportText(result Result) string {
	if result.Skipped {
		return result.Reason + "\n"
	}
	var b strings.Builder
	b.WriteString(result.Config.ConfigSummary() + "\n")
	b.WriteString(result.Map.Summary() + "\n")
	if result.Report.Derived > 0 {
		fmt.Fprintf(&b, "derivation: %d symbols over %d files (%s)\n",
			result.Report.Derived, len(result.Derived.Sources), strings.Join(result.Derived.Sources, ", "))
	}
	errors := 0
	warnings := 0
	for _, f := range result.Report.Findings {
		if f.Severity == SeverityError {
			errors++
		} else {
			warnings++
		}
	}
	fmt.Fprintf(&b, "findings: %d error(s), %d warning(s)\n", errors, warnings)
	sorted := append([]Finding{}, result.Report.Findings...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Code < sorted[j].Code })
	for _, f := range sorted {
		node := ""
		if f.Node != "" {
			node = " [" + f.Node + "]"
		}
		fmt.Fprintf(&b, "  %s %s%s: %s\n", f.Severity, f.Code, node, f.Message)
		if f.Hint != "" {
			fmt.Fprintf(&b, "      → %s\n", f.Hint)
		}
	}
	if len(result.Undeclared()) > 0 {
		fmt.Fprintf(&b, "undeclared candidates: %s\n", FormatUndeclared(result.Undeclared(), 10))
	}
	if len(result.Projections) > 0 {
		b.WriteString("projections:\n")
		for _, p := range result.Projections {
			fmt.Fprintf(&b, "  %-9s %-34s ~%d tokens\n", p.Target, p.Path, p.Tokens)
		}
	}
	return b.String()
}

// Undeclared returns the derived elements the map does not describe.
func (r Result) Undeclared() []string {
	var out []string
	for _, f := range r.Report.Findings {
		if f.Code == "undeclared-element" {
			out = append(out, f.Message)
		}
	}
	return out
}

// ExpandPattern resolves a derivation pattern, supporting `**` for "any depth".
//
// `filepath.Glob` has no `**`, and the alternative — hard-coding per-language
// extensions into the core — would make the neutral seam language-specific. So
// the core understands one extra construct and nothing about Go.
func ExpandPattern(root, pattern string) ([]string, error) {
	if !strings.Contains(pattern, "**") {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return nil, fmt.Errorf("uimap: bad pattern %q: %w", pattern, err)
		}
		return matches, nil
	}
	head, tail, _ := strings.Cut(pattern, "**/")
	base := filepath.Join(root, head)
	var out []string
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			// A missing base directory is an empty match, not a failure: a
			// project may declare a scope before creating it.
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		matched, matchErr := filepath.Match(tail, entry.Name())
		if matchErr != nil {
			return matchErr
		}
		if matched {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("uimap: walk %q: %w", pattern, err)
	}
	sort.Strings(out)
	return out, nil
}
