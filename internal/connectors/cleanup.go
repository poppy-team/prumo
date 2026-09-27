package connectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/harness/doccompile"
	"github.com/raillen/prumo/internal/install"
)

// SaveCleanup writes the cleanup manifest for a connector into PRUMO_HOME.
func SaveCleanup(home, connectorID string, manifest install.CleanupManifest) error {
	path := install.CleanupPath(home, connectorID)
	if manifest.ProjectRoot != "" {
		projPath := install.CleanupPathForProject(home, connectorID, manifest.ProjectRoot)
		if err := os.MkdirAll(filepath.Dir(projPath), 0755); err == nil {
			data, _ := json.MarshalIndent(manifest, "", "  ")
			_ = os.WriteFile(projPath, append(data, '\n'), 0644)
		}
	}
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
func LoadCleanup(home, connectorID string, projectRoot ...string) (*install.CleanupManifest, error) {
	path := install.CleanupPath(home, connectorID)
	if len(projectRoot) > 0 && projectRoot[0] != "" {
		projPath := install.CleanupPathForProject(home, connectorID, projectRoot[0])
		if data, err := os.ReadFile(projPath); err == nil {
			var manifest install.CleanupManifest
			if json.Unmarshal(data, &manifest) == nil {
				return &manifest, nil
			}
		}
	}
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

	removed, leftovers := install.RemoveManagedPaths(manifest.CreatedPaths)

	// Surgical removal of managed fragments from pre-existing files without deleting the user's file
	for _, fragment := range manifest.ManagedFragments {
		parts := strings.SplitN(fragment, ":", 2)
		if len(parts) != 2 {
			continue
		}
		filePath, regionID := parts[0], parts[1]
		if !filepath.IsAbs(filePath) && projectRoot != "" {
			filePath = filepath.Join(projectRoot, filePath)
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		cleaned, err := doccompile.RemoveRegion(string(data), regionID)
		if err != nil {
			continue
		}
		if strings.TrimSpace(cleaned) == "" {
			if err := os.Remove(filePath); err == nil {
				removed = append(removed, filePath)
			} else {
				leftovers = append(leftovers, filePath)
			}
		} else {
			if err := os.WriteFile(filePath, []byte(cleaned), 0644); err == nil {
				removed = append(removed, filePath+" ("+regionID+")")
			} else {
				leftovers = append(leftovers, filePath)
			}
		}
	}

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
	if projectRoot != "" {
		_ = os.Remove(install.CleanupPathForProject(home, connectorID, projectRoot))
	}

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
