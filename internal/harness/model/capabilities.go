package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CapabilitySet is what a model can do, as far as anyone has declared.
//
// It is **declared**, not probed: no provider lists its models' features in a
// way a client can read, so the harness reports what somebody wrote down and
// says plainly when nobody did. An absent capability is not a denial — it is
// the absence of a claim, which is why every field is a pointer-free `false`
// next to a `Declared` flag on the record rather than an assertion of inability.
type CapabilitySet struct {
	Text          bool `json:"text,omitempty"`
	Vision        bool `json:"vision,omitempty"`
	Reasoning     bool `json:"reasoning,omitempty"`
	Tools         bool `json:"tools,omitempty"`
	Audio         bool `json:"audio,omitempty"`
	ContextTokens int  `json:"context_tokens,omitempty"`
}

// Declarations is the workspace's own answer to "what can these models do".
type Declarations struct {
	Version int                      `json:"version"`
	Models  map[string]CapabilitySet `json:"models"`
}

// DefaultDeclarationsPath is where a workspace writes what its models can do.
const DefaultDeclarationsPath = ".prumo/models.json"

// ModelInfo is one model as the daemon reports it.
//
// A model the workspace never declared is reported with `Declared: false` and an
// empty set: a client can then say "not declared" instead of drawing a model
// without eyes as one that cannot see.
type ModelInfo struct {
	ID           string        `json:"id"`
	Declared     bool          `json:"declared"`
	Capabilities CapabilitySet `json:"capabilities"`
}

// LoadDeclarations reads a workspace's declarations, if it has any.
//
// A missing file is the ordinary case, not a failure: the harness serves models
// it was told about and claims nothing about them.
func LoadDeclarations(workspace string) (Declarations, error) {
	path := filepath.Join(workspace, DefaultDeclarationsPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Declarations{}, nil
		}
		return Declarations{}, err
	}
	var doc Declarations
	if err := json.Unmarshal(data, &doc); err != nil {
		return Declarations{}, fmt.Errorf("%s: %w", path, err)
	}
	return doc, nil
}

// Describe pairs the models a provider serves with what the workspace declared
// about them, in the order the provider gave them.
func Describe(ids []string, declared Declarations) []ModelInfo {
	out := make([]ModelInfo, 0, len(ids))
	for _, id := range ids {
		info := ModelInfo{ID: id}
		if set, ok := declared.Models[id]; ok {
			info.Declared = true
			info.Capabilities = set
		}
		out = append(out, info)
	}
	return out
}

// Features is the declared set as a short list of words, for a reader.
//
// The order is fixed so the same model always reads the same way, and a model
// nobody declared returns an empty list rather than a list of denials.
func (c CapabilitySet) Features() []string {
	features := make([]string, 0, 5)
	for _, feature := range []struct {
		name string
		set  bool
	}{
		{"text", c.Text},
		{"reasoning", c.Reasoning},
		{"vision", c.Vision},
		{"tools", c.Tools},
		{"audio", c.Audio},
	} {
		if feature.set {
			features = append(features, feature.name)
		}
	}
	return features
}
