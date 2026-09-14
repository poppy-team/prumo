// Package doccompile implements the deterministic Documentation Compiler
// baseline: Knowledge IR -> Document IR -> dependency DAG -> incremental
// evaluator -> renderer -> canonical serializer -> atomic writer, with
// stable IDs, SHA-256 fingerprints, CAS/no-op writes and manifests.
package doccompile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Node is one pure build unit.
type Node struct {
	ID      string   `json:"id"`
	Inputs  []string `json:"inputs"`
	Content string   `json:"content"`
}

// Artifact is one rendered output.
type Artifact struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
	Bytes       int    `json:"bytes"`
	NoOp        bool   `json:"no_op"`
}

// Compiler holds the DAG + content-addressed cache.
type Compiler struct {
	Nodes map[string]Node
	Cache map[string]string // nodeID -> fingerprint
}

func New() *Compiler { return &Compiler{Nodes: map[string]Node{}, Cache: map[string]string{}} }

func (c *Compiler) Add(n Node) { c.Nodes[n.ID] = n }

// Order topologically sorts the DAG; cycles are errors.
func (c *Compiler) Order() ([]string, error) {
	indeg := map[string]int{}
	for id := range c.Nodes {
		indeg[id] = 0
	}
	for _, n := range c.Nodes {
		for _, dep := range n.Inputs {
			if _, ok := c.Nodes[dep]; !ok {
				return nil, fmt.Errorf("unknown dep %s of %s", dep, n.ID)
			}
			indeg[n.ID]++
		}
	}
	queue := []string{}
	for id, d := range indeg {
		if d == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)
	var out []string
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		out = append(out, id)
		for nid, n := range c.Nodes {
			for _, dep := range n.Inputs {
				if dep == id {
					indeg[nid]--
					if indeg[nid] == 0 {
						queue = append(queue, nid)
					}
				}
			}
		}
		sort.Strings(queue)
	}
	if len(out) != len(c.Nodes) {
		return nil, fmt.Errorf("dependency cycle detected")
	}
	return out, nil
}

func fingerprint(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Render concatenates node contents in DAG order (deterministic baseline).
func (c *Compiler) Render() (string, []string, error) {
	order, err := c.Order()
	if err != nil {
		return "", nil, err
	}
	out := ""
	for _, id := range order {
		out += c.Nodes[id].Content + "\n"
	}
	return out, order, nil
}

// Write atomically writes content to path with CAS: unchanged content is a no-op.
func Write(path, content string) (Artifact, error) {
	fp := fingerprint(content)
	if old, err := os.ReadFile(path); err == nil && fingerprint(string(old)) == fp {
		return Artifact{Path: path, Fingerprint: fp, Bytes: len(content), NoOp: true}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Artifact{}, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return Artifact{}, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return Artifact{}, err
	}
	return Artifact{Path: path, Fingerprint: fp, Bytes: len(content)}, nil
}

// Manifest describes one build for provenance.
type Manifest struct {
	Order       []string   `json:"order"`
	Fingerprint string     `json:"fingerprint"`
	Artifacts   []Artifact `json:"artifacts"`
}

func (m Manifest) Save(path string) error {
	data, _ := json.MarshalIndent(m, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
