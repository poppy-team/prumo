// Gate policies (GAP-002 remainder): declarative completion gates beyond
// --strict, evaluated against tool reports and budget usage. Policies load
// from JSON (--gates policy.json); results use the protocol gates shape.
package runlayer

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/raillen/prumo/internal/protocol/gates"
)

// GatePolicy is one declarative completion rule.
type GatePolicy struct {
	Name  string         `json:"name"`
	Kind  string         `json:"kind"` // require-test-pass|forbid-tool-failure|max-tool-calls|max-tokens
	Limit float64        `json:"limit,omitempty"`
	Extra map[string]any `json:"extra,omitempty"`
}

// LoadGatePolicies reads a JSON list of policies.
func LoadGatePolicies(path string) ([]GatePolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []GatePolicy
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("gates file: %w", err)
	}
	return out, nil
}

// EvaluateGates runs every policy; unknown kinds fail closed.
func EvaluateGates(policies []GatePolicy, reports []ToolReport, usage map[string]float64) []gates.Result {
	out := make([]gates.Result, 0, len(policies))
	pass := func(name string) {
		out = append(out, gates.Result{ID: name, Status: "pass"})
	}
	fail := func(name, reason string) {
		out = append(out, gates.Result{ID: name, Status: "fail", Reason: reason})
	}
	for _, p := range policies {
		if p.Name == "" {
			fail("(unnamed)", "gate requires a name")
			continue
		}
		switch p.Kind {
		case "require-test-pass":
			ok := false
			for _, r := range reports {
				if r.Name == "test.run" && r.ExitCode == 0 {
					ok = true
				}
			}
			if ok {
				pass(p.Name)
			} else {
				fail(p.Name, "no passing test.run observed")
			}
		case "forbid-tool-failure":
			bad := ""
			for _, r := range reports {
				if r.ExitCode != 0 {
					bad = r.Name
				}
			}
			if bad == "" {
				pass(p.Name)
			} else {
				fail(p.Name, "tool failed: "+bad)
			}
		case "max-tool-calls":
			n := 0.0
			for range reports {
				n++
			}
			if n <= p.Limit {
				pass(p.Name)
			} else {
				fail(p.Name, fmt.Sprintf("%.0f calls over limit %.0f", n, p.Limit))
			}
		case "max-tokens":
			if usage["tokens"] <= p.Limit {
				pass(p.Name)
			} else {
				fail(p.Name, fmt.Sprintf("%.0f tokens over limit %.0f", usage["tokens"], p.Limit))
			}
		default:
			fail(p.Name, "unknown gate kind "+p.Kind)
		}
	}
	return out
}

// GatesQualityGate adapts policies to a Runner gate: any fail blocks.
func GatesQualityGate(policies []GatePolicy, reports func() []ToolReport, usage func() map[string]float64) func() error {
	return func() error {
		for _, r := range EvaluateGates(policies, reports(), usage()) {
			if r.Status != "pass" {
				return fmt.Errorf("gate %s: %s", r.ID, r.Reason)
			}
		}
		return nil
	}
}
