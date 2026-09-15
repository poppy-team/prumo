package agentsurface

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/raillen/prumo/internal/harness/doccompile"
	"github.com/raillen/prumo/internal/harness/model"
)

// DefaultOutDir receives standalone vendor surfaces. It is gitignored runtime
// state, so a vendor projection never becomes a canonical repository artifact.
const DefaultOutDir = ".prumo/runtime/agents"

// InstructionFiles are the agent-facing documents whose code references are
// validated continuously (context-rot guard, W16.14).
var InstructionFiles = []string{
	"AGENTS.md", "ENTRYPOINT.md", "FRAMEWORK.md", "README.md", "docs/PRUMO.md",
}

// Fingerprint is the content hash recorded in the generated marker.
func Fingerprint(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])[:16]
}

// TokenEstimate reuses the harness estimator so instruction budgets and context
// budgets are measured with the same documented heuristic.
func TokenEstimate(text string) int { return model.EstimateTokens(text, "") }

// Report is the verification result for the agent instruction surfaces.
type Report struct {
	IRVersion int       `json:"ir_version"`
	Adapters  []string  `json:"adapters"`
	Scopes    []string  `json:"scopes"`
	Surfaces  int       `json:"surfaces"`
	Findings  []Finding `json:"findings"`
	OK        bool      `json:"ok"`
}

// RenderAll compiles every adapter surface from the IR without touching disk.
func RenderAll(ir IR) ([]Surface, error) {
	registry := Adapters()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)

	var surfaces []Surface
	for _, name := range names {
		for _, scope := range ir.Scopes() {
			surface, ok, err := registry[name].Render(ir, scope)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			surface.Tokens = TokenEstimate(surface.Content)
			surfaces = append(surfaces, surface)
		}
	}
	sort.Slice(surfaces, func(i, j int) bool {
		if surfaces[i].Adapter != surfaces[j].Adapter {
			return surfaces[i].Adapter < surfaces[j].Adapter
		}
		return surfaces[i].Scope < surfaces[j].Scope
	})
	return surfaces, nil
}

func budgetFor(ir IR, scope string) int {
	if NormalScope(scope) == "" {
		return ir.TokenBudget.Root
	}
	return ir.TokenBudget.Scope
}

func checkBudget(ir IR, surfaces []Surface) []Finding {
	var findings []Finding
	for _, s := range surfaces {
		cap := budgetFor(ir, s.Scope)
		if cap <= 0 || s.Tokens <= cap {
			continue
		}
		findings = append(findings, Finding{
			Kind: "token-budget-exceeded", Target: s.Path, Value: scopeLabel(s.Scope),
			Detail: fmt.Sprintf("%s surface uses %d tokens, budget is %d", s.Adapter, s.Tokens, cap),
		})
	}
	return findings
}

func conflictFindings(ir IR) []Finding {
	var findings []Finding
	for _, scope := range ir.Scopes() {
		for _, c := range ir.Conflicts(scope) {
			findings = append(findings, Finding{
				Kind: c.Kind, Target: describeScope(NormalScope(scope)), Value: c.RuleID, Detail: c.Detail,
			})
		}
	}
	return findings
}

func validationFindings(ir IR) []Finding {
	var findings []Finding
	for _, p := range ir.Validate() {
		findings = append(findings, Finding{Kind: p.Kind, Target: IRPath, Value: p.RuleID, Detail: p.Detail})
	}
	return findings
}

// CheckRegion compares a managed region against its rendered body.
func CheckRegion(root string, surface Surface) []Finding {
	path := filepath.Join(root, filepath.FromSlash(surface.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		return []Finding{{
			Kind: "surface-missing", Target: surface.Path, Value: surface.Adapter,
			Detail: "curated file with a managed agent region does not exist",
		}}
	}
	body, err := doccompile.ExtractRegion(string(data), surface.Region)
	if err != nil {
		return []Finding{{
			Kind: "surface-region-missing", Target: surface.Path, Value: surface.Region,
			Detail: err.Error(),
		}}
	}
	regionBody, recorded, ok := StripProvenance(body)
	if !ok {
		return []Finding{{
			Kind: "surface-unmanaged", Target: surface.Path, Value: surface.Region,
			Detail: "managed region has no generated marker; run docs agents build",
		}}
	}
	rendered, renderedFingerprint, _ := StripProvenance(surface.Content)
	if recorded != renderedFingerprint || normalizeRegion(regionBody) != normalizeRegion(rendered) {
		return []Finding{{
			Kind: "surface-stale", Target: surface.Path, Value: surface.Adapter,
			Detail: fmt.Sprintf("region %s is out of date (recorded %s, rendered %s)", surface.Region, recorded, renderedFingerprint),
		}}
	}
	return nil
}

func normalizeRegion(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

// CheckStandaloneSurface verifies a committed vendor surface when one exists.
// Absent surfaces are not findings: vendor files are optional projections.
func CheckStandaloneSurface(root string, surface Surface) []Finding {
	path := filepath.Join(root, filepath.FromSlash(surface.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	_, recorded, ok := StripProvenance(string(data))
	if !ok {
		return []Finding{{
			Kind: "surface-unmanaged", Target: surface.Path, Value: surface.Adapter,
			Detail: "committed vendor surface has no generated marker",
		}}
	}
	if recorded != surface.Fingerprint {
		return []Finding{{
			Kind: "surface-stale", Target: surface.Path, Value: surface.Adapter,
			Detail: fmt.Sprintf("committed surface does not match the compiled IR (recorded %s, rendered %s)", recorded, surface.Fingerprint),
		}}
	}
	return nil
}

// CheckContextRot validates every code reference carried by the curated agent
// instruction documents and by the compiled surfaces (W16.10).
func CheckContextRot(root string) ([]Finding, error) {
	var referencers []Referencer
	for _, rel := range InstructionFiles {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		referencers = append(referencers, Referencer{Target: rel, Text: string(data)})
	}
	ir, err := LoadIR(root)
	if err != nil {
		return nil, err
	}
	surfaces, err := RenderAll(ir)
	if err != nil {
		return nil, err
	}
	for _, s := range surfaces {
		referencers = append(referencers, Referencer{Target: s.Path, Text: s.Content})
	}
	return ValidateReferences(root, referencers), nil
}

// Verify checks the IR, its scopes, budgets, conflicts, the managed regions and
// every reference. It is the context-rot CI gate (W16.14).
func Verify(root string) (Report, error) {
	ir, err := LoadIR(root)
	if err != nil {
		return Report{}, err
	}
	surfaces, err := RenderAll(ir)
	if err != nil {
		return Report{}, err
	}
	registry := Adapters()
	adapters := make([]string, 0, len(registry))
	for name := range registry {
		adapters = append(adapters, name)
	}
	sort.Strings(adapters)

	report := Report{
		IRVersion: ir.Version,
		Adapters:  adapters,
		Scopes:    ir.Scopes(),
		Surfaces:  len(surfaces),
		Findings:  []Finding{},
	}
	report.Findings = append(report.Findings, validationFindings(ir)...)
	report.Findings = append(report.Findings, conflictFindings(ir)...)
	report.Findings = append(report.Findings, checkBudget(ir, surfaces)...)
	for _, s := range surfaces {
		if s.Region != "" {
			report.Findings = append(report.Findings, CheckRegion(root, s)...)
			continue
		}
		report.Findings = append(report.Findings, CheckStandaloneSurface(root, s)...)
	}
	rot, err := CheckContextRot(root)
	if err != nil {
		return Report{}, err
	}
	report.Findings = append(report.Findings, rot...)
	sortFindings(report.Findings)
	report.OK = len(report.Findings) == 0
	return report, nil
}

// BuildResult records what a build wrote.
type BuildResult struct {
	IRVersion int      `json:"ir_version"`
	Surfaces  int      `json:"surfaces"`
	Written   []string `json:"written"`
	Unchanged []string `json:"unchanged"`
	OutDir    string   `json:"out_dir"`
	Manifest  string   `json:"manifest,omitempty"`
}

// Build regenerates every surface. Managed regions are updated in place;
// standalone vendor surfaces are written under outDir (gitignored runtime).
func Build(root, outDir string) (BuildResult, error) {
	if outDir == "" {
		outDir = DefaultOutDir
	}
	ir, err := LoadIR(root)
	if err != nil {
		return BuildResult{}, err
	}
	surfaces, err := RenderAll(ir)
	if err != nil {
		return BuildResult{}, err
	}
	result := BuildResult{IRVersion: ir.Version, Surfaces: len(surfaces), OutDir: outDir,
		Written: []string{}, Unchanged: []string{}}
	for _, s := range surfaces {
		target := s.LocalPath(outDir)
		abs := filepath.Join(root, filepath.FromSlash(target))
		if s.Region != "" {
			data, err := os.ReadFile(abs)
			if err != nil {
				return BuildResult{}, fmt.Errorf("managed region target %s: %w", target, err)
			}
			updated, err := doccompile.ReplaceRegion(string(data), s.Region, s.Content)
			if err != nil {
				return BuildResult{}, err
			}
			if updated == string(data) {
				result.Unchanged = append(result.Unchanged, target)
				continue
			}
			if _, err := doccompile.Write(abs, updated); err != nil {
				return BuildResult{}, err
			}
			result.Written = append(result.Written, target)
			continue
		}
		if existing, err := os.ReadFile(abs); err == nil && string(existing) == s.Content {
			result.Unchanged = append(result.Unchanged, target)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return BuildResult{}, err
		}
		if _, err := doccompile.Write(abs, s.Content); err != nil {
			return BuildResult{}, err
		}
		result.Written = append(result.Written, target)
	}
	manifestPath := filepath.Join(root, filepath.FromSlash(outDir), "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return BuildResult{}, err
	}
	manifest := Manifest{IRVersion: ir.Version, Generated: IRPath, Surfaces: surfaces}
	if err := manifest.Save(manifestPath); err != nil {
		return BuildResult{}, err
	}
	result.Manifest = filepath.ToSlash(filepath.Join(outDir, "manifest.json"))
	sort.Strings(result.Written)
	sort.Strings(result.Unchanged)
	return result, nil
}

// Manifest is the sidecar describing the compiled surfaces (W16.9).
type Manifest struct {
	IRVersion int       `json:"ir_version"`
	Generated string    `json:"generated_from"`
	Surfaces  []Surface `json:"surfaces"`
}

// Save writes the manifest sidecar through the deterministic writer.
func (m Manifest) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	_, err = doccompile.Write(path, string(append(data, '\n')))
	return err
}

// AffectedSurfaces lists the compiled surfaces a set of changed paths
// invalidates: a change to a rule source, to the IR, or to anything inside a
// surface's scope (W16.15).
func AffectedSurfaces(ir IR, changed []string) []Surface {
	surfaces, err := RenderAll(ir)
	if err != nil {
		return nil
	}
	var affected []Surface
	for _, s := range surfaces {
		if surfaceAffected(ir, s, changed) {
			affected = append(affected, s)
		}
	}
	return affected
}

func surfaceAffected(ir IR, s Surface, changed []string) bool {
	for _, raw := range changed {
		path := strings.TrimSpace(filepath.ToSlash(raw))
		if path == "" {
			continue
		}
		if path == IRPath || path == s.Path {
			return true
		}
		if s.Scope != "" && ScopeContains(s.Scope, path) {
			return true
		}
		// Only the sources of the rules this surface actually renders count; the
		// aggregate provenance list would otherwise over-report every change.
		for _, rule := range ir.Rules {
			if !containsString(s.Rules, rule.ID) {
				continue
			}
			for _, source := range rule.Sources {
				source = strings.TrimSuffix(filepath.ToSlash(source), "/")
				if path == source || strings.HasPrefix(path, source+"/") {
					return true
				}
			}
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// Explain returns one rendered surface by adapter and scope.
func Explain(ir IR, adapter, scope string) (Surface, error) {
	registry := Adapters()
	a, ok := registry[adapter]
	if !ok {
		return Surface{}, fmt.Errorf("unknown adapter: %s", adapter)
	}
	surface, ok, err := a.Render(ir, scope)
	if err != nil {
		return Surface{}, err
	}
	if !ok {
		return Surface{}, fmt.Errorf("no rules apply to %s for adapter %s", scopeLabel(scope), adapter)
	}
	surface.Tokens = TokenEstimate(surface.Content)
	return surface, nil
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		if findings[i].Target != findings[j].Target {
			return findings[i].Target < findings[j].Target
		}
		return findings[i].Value < findings[j].Value
	})
}

// JSON renders a report deterministically for CLI output.
func (r Report) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
