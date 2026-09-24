package connectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/raillen/prumo/internal/install"
)

// SaveCleanup writes the cleanup manifest for a connector into PRUMO_HOME,
// under the project it describes.
//
// The project is part of the path because one file per connector cannot hold
// two projects: installing into a second project overwrote the first project's
// record, and uninstalling then cleaned up the wrong one and orphaned the
// other's files (GAP-137).
func SaveCleanup(home, connectorID, projectRoot string, manifest install.CleanupManifest) error {
	path := install.CleanupPath(home, connectorID, projectRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return err
	}
	// The record exists; only now is the index allowed to name it.
	return install.RecordCleanup(home, connectorID, projectRoot)
}

// LoadCleanup reads the cleanup manifest for a connector in one project.
func LoadCleanup(home, connectorID, projectRoot string) (*install.CleanupManifest, error) {
	path := install.CleanupPath(home, connectorID, projectRoot)
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
	manifest, err := LoadCleanup(home, connectorID, projectRoot)
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

	// Remove this project's cleanup record and its index entry. Other projects
	// that installed the same connector keep theirs: cleaning one project must
	// not forget that another one was installed (GAP-137).
	cleanupPath := install.CleanupPath(home, connectorID, projectRoot)
	_ = os.Remove(cleanupPath)
	_ = install.ForgetCleanup(home, connectorID, projectRoot)

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
