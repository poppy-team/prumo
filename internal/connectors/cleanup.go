package connectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/raillen/prumo/internal/install"
)

// SaveCleanup writes the cleanup manifest for a connector into PRUMO_HOME.
func SaveCleanup(home, connectorID string, manifest install.CleanupManifest) error {
	path := install.CleanupPath(home, connectorID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// LoadCleanup reads the cleanup manifest for a connector from PRUMO_HOME.
func LoadCleanup(home, connectorID string) (*install.CleanupManifest, error) {
	path := install.CleanupPath(home, connectorID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cleanup manifest not found for connector %q: %w", connectorID, err)
	}
	var manifest install.CleanupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("corrupt cleanup manifest for connector %q: %w", connectorID, err)
	}
	return &manifest, nil
}

// ExecuteCleanup removes files managed by the connector and updates installation state.
func ExecuteCleanup(home, connectorID string, projectRoot string, pruneDirs ...string) (*UninstallResult, error) {
	manifest, err := LoadCleanup(home, connectorID)
	if err != nil {
		return nil, err
	}

	removed, leftovers := install.RemoveManagedPaths(home, manifest.CreatedPaths)

	// Prune directories if they become empty
	for _, dir := range pruneDirs {
		target := dir
		if !filepath.IsAbs(target) && projectRoot != "" {
			target = filepath.Join(projectRoot, dir)
		}
		_ = os.Remove(target) // succeeds only if directory is empty
	}

	// Remove cleanup manifest
	cleanupPath := install.CleanupPath(home, connectorID)
	_ = os.Remove(cleanupPath)

	// Update installation manifest in PRUMO_HOME
	instManifest, err := install.LoadManifest(home)
	if err == nil {
		delete(instManifest.Connectors, connectorID)
		_ = install.SaveManifest(home, instManifest)
	}

	return &UninstallResult{
		Connector: connectorID,
		Removed:   removed,
		Leftovers: leftovers,
		Clean:     len(leftovers) == 0,
	}, nil
}
