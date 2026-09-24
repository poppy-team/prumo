package install

import (
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
func CleanupPath(home, connector string) string {
	return filepath.Join(home, "connectors", connector, "cleanup.json")
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
