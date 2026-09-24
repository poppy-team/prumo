package install

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Manifest struct {
	PrumoVersion           string            `json:"prumo_version"`
	BinaryPath             string            `json:"binary_path"`
	Connectors             map[string]string `json:"connectors"`
	CreatedPaths           []string          `json:"created_paths"`
	ManagedConfigFragments []string          `json:"managed_config_fragments"`
}

type CleanupManifest struct {
	Connector        string   `json:"connector"`
	Scope            string   `json:"scope"`
	CreatedPaths     []string `json:"created_paths"`
	ManagedFragments []string `json:"managed_fragments"`
	Backups          []string `json:"backups"`
}

func HomeDir(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	if value := os.Getenv("PRUMO_HOME"); value != "" {
		return filepath.Abs(value)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".prumo"), nil
}

func ManifestPath(home string) string { return filepath.Join(home, "config", "installation.json") }

// CleanupPath is where one connector's cleanup record lives for one project.
//
// The project is part of the path because the record was one file per
// connector, with no project in it. Installing a connector into a second project
// overwrote the first project's record, so uninstalling cleaned up whichever
// project was installed last and left the other one's files behind with nothing
// pointing at them (GAP-137).
//
// The project is named by a digest of its absolute path: a readable name would
// have to be escaped, sanitised and kept unique, and a path segment that
// changes is a path that leaks a user's directory layout into a filename.
func CleanupPath(home, connector, projectRoot string) string {
	return filepath.Join(home, "connectors", connector, ScopeDir(projectRoot), "cleanup.json")
}

// ScopeDir is the per-project directory name under a connector.
func ScopeDir(projectRoot string) string {
	if projectRoot == "" {
		return "global"
	}
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		abs = projectRoot
	}
	sum := sha256.Sum256([]byte(filepath.Clean(abs)))
	return "project-" + hex.EncodeToString(sum[:8])
}

// CleanupIndexPath lists every cleanup record, so an uninstall that is not told
// which project it is cleaning can still find them all.
func CleanupIndexPath(home string) string {
	return filepath.Join(home, "connectors", "cleanup-index.json")
}

// CleanupIndexEntry is one record in the index.
type CleanupIndexEntry struct {
	Connector   string `json:"connector"`
	ProjectRoot string `json:"project_root"`
	Path        string `json:"path"`
}

// CleanupIndex is every recorded cleanup, across connectors and projects.
type CleanupIndex struct {
	Version int                 `json:"version"`
	Entries []CleanupIndexEntry `json:"entries"`
}

// LoadCleanupIndex reads the index, returning an empty one when absent.
func LoadCleanupIndex(home string) (CleanupIndex, error) {
	index := CleanupIndex{Version: 1}
	data, err := os.ReadFile(CleanupIndexPath(home))
	if err != nil {
		if os.IsNotExist(err) {
			return index, nil
		}
		return index, err
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return CleanupIndex{Version: 1}, err
	}
	return index, nil
}

// RecordCleanup adds or replaces one entry in the index.
//
// The index is written after the record it names. An index entry pointing at a
// file that was never written is worse than a missing entry: uninstall would try
// to clean up something that is not there and report the failure as if the
// cleanup itself had failed.
func RecordCleanup(home, connector, projectRoot string) error {
	path := CleanupPath(home, connector, projectRoot)
	index, err := LoadCleanupIndex(home)
	if err != nil {
		return err
	}
	entry := CleanupIndexEntry{Connector: connector, ProjectRoot: projectRoot, Path: path}
	for i := range index.Entries {
		if index.Entries[i].Connector == connector && index.Entries[i].ProjectRoot == projectRoot {
			index.Entries[i] = entry
			if err := writeCleanupIndex(home, index); err != nil {
				return err
			}
			return nil
		}
	}
	index.Entries = append(index.Entries, entry)
	sort.Slice(index.Entries, func(i, j int) bool {
		if index.Entries[i].Connector != index.Entries[j].Connector {
			return index.Entries[i].Connector < index.Entries[j].Connector
		}
		return index.Entries[i].ProjectRoot < index.Entries[j].ProjectRoot
	})
	return writeCleanupIndex(home, index)
}

func writeCleanupIndex(home string, index CleanupIndex) error {
	path := CleanupIndexPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func LoadManifest(home string) (Manifest, error) {
	var manifest Manifest
	manifest.Connectors = map[string]string{}
	data, err := os.ReadFile(ManifestPath(home))
	if err != nil {
		if os.IsNotExist(err) {
			return manifest, nil
		}
		return manifest, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, err
	}
	if manifest.Connectors == nil {
		manifest.Connectors = map[string]string{}
	}
	return manifest, nil
}

func SaveManifest(home string, manifest Manifest) error {
	if manifest.Connectors == nil {
		manifest.Connectors = map[string]string{}
	}
	manifest.CreatedPaths = sortedUnique(manifest.CreatedPaths)
	manifest.ManagedConfigFragments = sortedUnique(manifest.ManagedConfigFragments)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := ManifestPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func RecordPaths(home string, paths ...string) error {
	manifest, err := LoadManifest(home)
	if err != nil {
		return err
	}
	manifest.CreatedPaths = append(manifest.CreatedPaths, paths...)
	return SaveManifest(home, manifest)
}

func sortedUnique(values []string) []string {
	unique := map[string]bool{}
	for _, value := range values {
		unique[value] = true
	}
	out := make([]string, 0, len(unique))
	for value := range unique {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func PurgeDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		target := filepath.Join(path, entry.Name())
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("failed to purge %s: %w", target, err)
		}
	}
	return nil
}

// ForgetCleanup drops one entry from the cleanup index.
//
// It is separate from RecordCleanup because uninstall and install have opposite
// failure modes: an install that cannot record itself should stop, while an
// uninstall that cannot update the index has still removed the files and should
// say so.
func ForgetCleanup(home, connector, projectRoot string) error {
	index, err := LoadCleanupIndex(home)
	if err != nil {
		return err
	}
	kept := make([]CleanupIndexEntry, 0, len(index.Entries))
	for _, entry := range index.Entries {
		if entry.Connector == connector && entry.ProjectRoot == projectRoot {
			continue
		}
		kept = append(kept, entry)
	}
	index.Entries = kept
	return writeCleanupIndex(home, index)
}
