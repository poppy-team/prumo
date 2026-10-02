package repositorypolicy

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
)

//go:embed default_policy.json
var defaultPolicyJSON []byte

// DefaultPolicy decodes the embedded default repository policy. Used to
// bootstrap a project that has not yet recorded one.
func DefaultPolicy() (Policy, error) {
	var p Policy
	if err := json.Unmarshal(defaultPolicyJSON, &p); err != nil {
		return Policy{}, err
	}
	return p, nil
}

// EnsurePolicyForInit writes a valid default repository policy into the root if
// the project does not already have one, returning whether it created a new
// one. AGENTS.md and the governance gate both assume the file exists, so a
// fresh project needs it by default.
func EnsurePolicyForInit(root string) (created bool, err error) {
	path := PolicyPath(root)
	if _, statErr := os.Stat(path); statErr == nil {
		return false, nil
	}
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
		return false, mkErr
	}
	p, err := DefaultPolicy()
	if err != nil {
		return false, err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, append(data, '\n'), 0o644)
}
