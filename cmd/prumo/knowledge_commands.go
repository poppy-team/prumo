package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/harness/knowledge"
	"github.com/raillen/prumo/internal/protocol"
)

// runtimeHarnessDir is where headless runs persist their per-run knowledge.
func runtimeHarnessDir(root string) string {
	return filepath.Join(root, ".prumo", "runtime", "harness")
}

// knowledgeManifestPath is where the derived index is written. It lives under
// runtime state: a manifest is a projection and never a canonical artifact.
func knowledgeManifestPath(root string) string {
	return filepath.Join(root, ".prumo", "runtime", "knowledge-manifest.json")
}

// loadRunStores reads every persisted per-run knowledge store, in filename
// order so the merge is deterministic.
func loadRunStores(root string) ([]*knowledge.Store, []string, error) {
	dir := runtimeHarnessDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "knowledge-") && strings.HasSuffix(name, ".json") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	stores := []*knowledge.Store{}
	for _, name := range names {
		store, err := knowledge.Load(filepath.Join(dir, name))
		if err != nil {
			return nil, nil, err
		}
		stores = append(stores, store)
	}
	return stores, names, nil
}

// runKnowledge dispatches `prumo knowledge <subcommand>`.
func runKnowledge(asJSON bool, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: knowledge requires a subcommand: manifest|ids")
		return exitUsage
	}
	path := pathFlag(args)
	var root string
	if path != "." {
		root = path
	} else {
		root = repoRoot()
	}
	switch args[0] {
	case "manifest":
		return runKnowledgeManifest(asJSON, args, root)
	case "ids":
		return runKnowledgeIDs(asJSON, root)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown knowledge subcommand: %s\n", args[0])
		return exitUsage
	}
}

func runKnowledgeManifest(asJSON bool, args []string, root string) int {
	stores, names, err := loadRunStores(root)
	if err != nil {
		return serviceError(asJSON, err)
	}
	merged := knowledge.Merge(stores...)
	manifest, problems := merged.Manifest(time.Now().UTC().Format(time.RFC3339), rootRevision(root))
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintf(os.Stderr, "error: %v\n", p)
		}
		return exitValidation
	}
	out := knowledgeManifestPath(root)
	if hasFlag(args, "--out") {
		for i, a := range args {
			if a == "--out" && i+1 < len(args) {
				out = args[i+1]
			}
		}
	}
	// The default destination is runtime state; only an explicit --out writes
	// outside it, so the derived index cannot silently become canonical.
	write := hasFlag(args, "--out") || !hasFlag(args, "--stdout")
	if write {
		if err := writeManifest(out, manifest); err != nil {
			return serviceError(asJSON, err)
		}
	}
	result := map[string]any{
		"manifest": manifest,
		"stores":   len(names),
		"out":      out,
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(result))
	}
	fmt.Printf("Knowledge manifest: %d units from %d run store(s)\n", manifest.Counts.Total, len(names))
	for _, kind := range sortedKeys(manifest.Counts.ByKind) {
		fmt.Printf("  %-16s %d\n", kind, manifest.Counts.ByKind[kind])
	}
	fmt.Printf("Written: %s (derived)\n", out)
	return exitOK
}

// runKnowledgeIDs reports records that still carry a pre-migration identifier.
// It is the migration report for W3.8: it never rewrites anything by itself.
func runKnowledgeIDs(asJSON bool, root string) int {
	stores, _, err := loadRunStores(root)
	if err != nil {
		return serviceError(asJSON, err)
	}
	merged := knowledge.Merge(stores...)
	unstable := merged.UnstableIDs()
	result := map[string]any{
		"total":        len(unstable),
		"unstable_ids": unstable,
		"note":         "pre-migration identifiers still resolve to their stable id; run a migration before removing them",
	}
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(result))
	}
	fmt.Printf("Unstable knowledge ids: %d\n", len(unstable))
	for _, id := range unstable {
		fmt.Printf("  %s\n", id)
	}
	return exitOK
}

func writeManifest(path string, manifest any) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// rootRevision names the revision the derived index was built from, so a
// manifest can be replayed against the tree that produced it.
func rootRevision(root string) string {
	cmd := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func sortedKeys(counts map[string]int) []string {
	out := make([]string, 0, len(counts))
	for k := range counts {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
