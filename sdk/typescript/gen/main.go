// Command gen emits sdk/typescript/protocol.d.ts from the checked-in IDL
// manifest (go:generate ./...). The .d.ts is generated, never hand-edited:
// freshness is enforced by TestGeneratedFresh.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root, err := repoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	out, err := Generate(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	dst := filepath.Join(root, "sdk", "typescript", "protocol.d.ts")
	if err := os.WriteFile(dst, []byte(out), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

// Generate renders protocol.d.ts from the checked-in manifest.
func Generate(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "schemas", "protocol-manifest.json"))
	if err != nil {
		return "", err
	}
	var doc struct {
		XManifest struct {
			Version       string   `json:"version"`
			MinCompatible string   `json:"min_compatible"`
			Transport     []string `json:"transport"`
			Ops           []struct {
				Name string   `json:"name"`
				Args []string `json:"args"`
			} `json:"ops"`
			Schemas []string `json:"schemas"`
			Errors  []string `json:"errors"`
		} `json:"x-manifest"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("// Generated from schemas/protocol-manifest.json — DO NOT EDIT.\n")
	b.WriteString("// Regenerate: go run ./sdk/typescript/gen\n\n")
	fmt.Fprintf(&b, "export const PROTOCOL_VERSION = %q;\n", doc.XManifest.Version)
	fmt.Fprintf(&b, "export const PROTOCOL_MIN_COMPATIBLE = %q;\n\n", doc.XManifest.MinCompatible)
	b.WriteString("export type OpName =\n")
	for i, op := range doc.XManifest.Ops {
		sep := " |"
		if i == len(doc.XManifest.Ops)-1 {
			sep = ";"
		}
		fmt.Fprintf(&b, "  | %q%s\n", op.Name, sep)
	}
	b.WriteString("\nexport interface OpArgs {\n")
	for _, op := range doc.XManifest.Ops {
		fmt.Fprintf(&b, "  %q: [%s];\n", op.Name, quoted(op.Args))
	}
	b.WriteString("}\n\n")
	b.WriteString("export interface RunStatus {\n  run_id: string;\n  status: string;\n  phase: string;\n  stop_reason: string;\n  active: boolean;\n}\n\n")
	b.WriteString("export interface AgentEvent {\n  id: string;\n  run_id: string;\n  turn_id?: string;\n  kind: string;\n  payload?: Record<string, unknown>;\n  created_at: string;\n}\n")
	return b.String(), nil
}

func quoted(args []string) string {
	q := make([]string, 0, len(args))
	for _, a := range args {
		q = append(q, fmt.Sprintf("%q", a))
	}
	return strings.Join(q, ", ")
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
