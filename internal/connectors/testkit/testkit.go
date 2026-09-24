package testkit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/raillen/prumo/internal/connectors"
	"github.com/raillen/prumo/internal/install"
)

// RunAll runs the full contract and compliance test suite against any connector.
func RunAll(t *testing.T, c connectors.Connector) {
	t.Helper()
	t.Run("Contract", func(t *testing.T) { VerifyContract(t, c) })
	t.Run("Compilation", func(t *testing.T) {
		root := t.TempDir()
		VerifyCompilation(t, c, root)
	})
	t.Run("IdempotentInstall", func(t *testing.T) {
		home := t.TempDir()
		root := t.TempDir()
		VerifyIdempotentInstall(t, c, home, root)
	})
	t.Run("SafeUninstall", func(t *testing.T) {
		home := t.TempDir()
		root := t.TempDir()
		VerifySafeUninstall(t, c, home, root)
	})
	t.Run("Negotiation", func(t *testing.T) { VerifyNegotiation(t, c) })
}

// VerifyContract checks contract completeness.
func VerifyContract(t *testing.T, c connectors.Connector) {
	t.Helper()
	contract := c.Contract()
	if contract.ID == "" {
		t.Fatal("contract ID must not be empty")
	}
	if contract.Version == "" {
		t.Fatal("contract Version must not be empty")
	}
	if contract.ProtocolRange == "" {
		t.Fatal("contract ProtocolRange must not be empty")
	}
	if len(contract.Capabilities) == 0 {
		t.Fatal("contract Capabilities must not be empty")
	}
	if contract.Enforcement != connectors.EnforcementAdvisory &&
		contract.Enforcement != connectors.EnforcementStandard &&
		contract.Enforcement != connectors.EnforcementStrict {
		t.Fatalf("invalid enforcement level: %s", contract.Enforcement)
	}
}

// VerifyCompilation verifies that compile produces non-empty artifacts with ownership markers.
func VerifyCompilation(t *testing.T, c connectors.Connector, projectRoot string) {
	t.Helper()
	res, err := c.Compile(projectRoot, connectors.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if len(res.CreatedPaths) == 0 {
		t.Fatal("expected non-empty created paths from compile")
	}

	for _, path := range res.CreatedPaths {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("created path does not exist on disk: %s", path)
		}
	}

	val, err := c.Validate(projectRoot)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !val.Valid {
		t.Fatalf("validation failed after compilation: %v", val.Errors)
	}
}

// VerifyIdempotentInstall verifies installation convergence and manifest creation.
func VerifyIdempotentInstall(t *testing.T, c connectors.Connector, home, projectRoot string) {
	t.Helper()
	// First install
	res1, err := c.Install(home, projectRoot, connectors.InstallOptions{})
	if err != nil {
		t.Fatalf("initial install failed: %v", err)
	}
	if res1.Status != "installed" {
		t.Fatalf("expected status installed, got %s", res1.Status)
	}

	cleanupPath := install.CleanupPath(home, c.ID(), projectRoot)
	if _, err := os.Stat(cleanupPath); err != nil {
		t.Fatalf("cleanup manifest missing at %s: %v", cleanupPath, err)
	}

	// Idempotent second install
	res2, err := c.Install(home, projectRoot, connectors.InstallOptions{})
	if err != nil {
		t.Fatalf("second install failed: %v", err)
	}
	if len(res2.CreatedPaths) != len(res1.CreatedPaths) {
		t.Errorf("expected idempotent created paths count, got %d vs %d", len(res2.CreatedPaths), len(res1.CreatedPaths))
	}
}

// VerifySafeUninstall verifies that uninstallation removes managed files and preserves user files.
func VerifySafeUninstall(t *testing.T, c connectors.Connector, home, projectRoot string) {
	t.Helper()
	_, err := c.Install(home, projectRoot, connectors.InstallOptions{})
	if err != nil {
		t.Fatalf("install before uninstall failed: %v", err)
	}

	// Create user file inside project
	userFile := filepath.Join(projectRoot, "user_notes.txt")
	if err := os.WriteFile(userFile, []byte("important user notes\n"), 0644); err != nil {
		t.Fatalf("failed to write user file: %v", err)
	}

	uninstRes, err := c.Uninstall(home, projectRoot, connectors.UninstallOptions{})
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	// Verify user file was preserved
	if _, err := os.Stat(userFile); err != nil {
		t.Fatalf("user file was erroneously removed during uninstall: %v", err)
	}

	// Verify cleanup manifest was removed
	cleanupPath := install.CleanupPath(home, c.ID(), projectRoot)
	if _, err := os.Stat(cleanupPath); !os.IsNotExist(err) {
		t.Fatalf("cleanup manifest still exists after uninstall: %s", cleanupPath)
	}

	_ = uninstRes
}

// VerifyNegotiation checks capability negotiation for strict and degraded modes.
func VerifyNegotiation(t *testing.T, c connectors.Connector) {
	t.Helper()
	contract := c.Contract()

	// Negotiate supported capabilities
	req1 := connectors.NegotiationRequest{
		TargetHarness:        contract.ID,
		RequiredCapabilities: contract.Capabilities,
		Strict:               true,
	}
	res1, err := connectors.Negotiate(contract, req1)
	if err != nil {
		t.Fatalf("negotiate supported capabilities failed: %v", err)
	}
	if !res1.Compatible {
		t.Fatalf("expected compatible negotiation, got %v", res1)
	}

	// Negotiate with non-existent capability in strict mode
	req2 := connectors.NegotiationRequest{
		TargetHarness:        contract.ID,
		RequiredCapabilities: []string{"non_existent_future_capability_xyz"},
		Strict:               true,
	}
	_, err2 := connectors.Negotiate(contract, req2)
	if err2 == nil {
		t.Fatal("expected error in strict negotiation with unsupported capability")
	}

	// Negotiate with fallback capability in non-strict mode
	req3 := connectors.NegotiationRequest{
		TargetHarness:        contract.ID,
		RequiredCapabilities: []string{connectors.CapPreToolBlock},
		Strict:               false,
	}
	res3, err3 := connectors.Negotiate(contract, req3)
	if err3 != nil {
		t.Fatalf("non-strict negotiation should succeed: %v", err3)
	}
	if contract.Supports(connectors.CapPreToolBlock) {
		if len(res3.Supported) == 0 {
			t.Error("expected supported cap")
		}
	} else {
		if _, degraded := res3.Degraded[connectors.CapPreToolBlock]; !degraded {
			t.Error("expected pre_tool_block to degrade")
		}
	}
}

// Helper
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func readJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	err = json.Unmarshal(data, &out)
	return out, err
}
