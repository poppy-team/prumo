package opencode_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/raillen/prumo/internal/connectors"
	"github.com/raillen/prumo/internal/connectors/opencode"
	"github.com/raillen/prumo/internal/connectors/testkit"
	"github.com/raillen/prumo/internal/install"
)

func TestOpenCodeTestKit(t *testing.T) {
	c := opencode.NewConnector()
	testkit.RunAll(t, c)
}

func TestOpenCodeContract(t *testing.T) {
	c := opencode.NewConnector()
	if c.ID() != "opencode" {
		t.Fatalf("expected ID opencode, got %s", c.ID())
	}
	contract := c.Contract()
	if contract.ID != "opencode" {
		t.Fatalf("expected contract ID opencode, got %s", contract.ID)
	}
	if contract.Enforcement != connectors.EnforcementStrict {
		t.Fatalf("expected strict enforcement, got %s", contract.Enforcement)
	}

	requiredCaps := []string{
		connectors.CapPreToolBlock,
		connectors.CapPostToolVerify,
		connectors.CapSessionHooks,
		connectors.CapIsolateSubagents,
		connectors.CapRestrictTools,
		connectors.CapNativePlugin,
		connectors.CapCommands,
		connectors.CapSubagents,
	}
	for _, cap := range requiredCaps {
		if !contract.Supports(cap) {
			t.Errorf("contract expected to support capability %s", cap)
		}
	}

	requiredHooks := []string{
		connectors.HookSessionStart,
		connectors.HookSessionEnd,
		connectors.HookToolBefore,
		connectors.HookToolAfter,
	}
	for _, hook := range requiredHooks {
		found := false
		for _, h := range contract.Hooks {
			if h == hook {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("contract expected to include hook %s", hook)
		}
	}
}

func TestOpenCodeCompileAndValidate(t *testing.T) {
	projectRoot := t.TempDir()
	c := opencode.NewConnector()

	res, err := c.Compile(projectRoot, connectors.CompileOptions{})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if res.Target != "opencode" {
		t.Fatalf("expected target opencode, got %s", res.Target)
	}
	if len(res.CreatedPaths) == 0 {
		t.Fatal("expected non-empty created paths")
	}

	// Validate compiled structure
	val, err := c.Validate(projectRoot)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !val.Valid {
		t.Fatalf("expected valid installation, got errors: %v", val.Errors)
	}

	// Verify plugin content
	pluginPath := filepath.Join(projectRoot, ".opencode", "plugins", "prumo.ts")
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("failed to read plugin: %v", err)
	}
	pluginStr := string(data)
	if !containsString(pluginStr, "validateToolExecution") {
		t.Errorf("plugin missing validateToolExecution")
	}
	if !containsString(pluginStr, "onSessionStart") {
		t.Errorf("plugin missing onSessionStart")
	}
	if !containsString(pluginStr, "onSessionEnd") {
		t.Errorf("plugin missing onSessionEnd")
	}

	// Verify ownership marker
	markerPath := filepath.Join(projectRoot, ".opencode", ".prumo-generated.json")
	mdata, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("failed to read marker: %v", err)
	}
	var marker map[string]any
	if err := json.Unmarshal(mdata, &marker); err != nil {
		t.Fatalf("marker unmarshal failed: %v", err)
	}
	if marker["managed"] != true || marker["target"] != "opencode" {
		t.Fatalf("unexpected marker contents: %v", marker)
	}

	// Verify opencode.json schema compliance (plugin must be array of strings)
	configPath := filepath.Join(projectRoot, ".opencode", "opencode.json")
	cfgData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read opencode.json: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		t.Fatalf("failed to unmarshal opencode.json: %v", err)
	}
	plugins, ok := cfg["plugin"].([]any)
	if !ok || len(plugins) == 0 {
		t.Fatalf("expected plugin to be a non-empty array in opencode.json, got: %T (%v)", cfg["plugin"], cfg["plugin"])
	}
	if plugins[0] != "plugins/prumo.ts" {
		t.Fatalf("expected plugin[0] to be plugins/prumo.ts, got: %v", plugins[0])
	}
}

func TestOpenCodeInstallAndUninstall(t *testing.T) {
	home := t.TempDir()
	projectRoot := t.TempDir()
	c := opencode.NewConnector()

	// Install
	instRes, err := c.Install(home, projectRoot, connectors.InstallOptions{})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if instRes.Status != "installed" {
		t.Fatalf("expected status installed, got %s", instRes.Status)
	}

	// Verify cleanup manifest exists
	cleanupPath := install.CleanupPath(home, "opencode", projectRoot)
	if _, err := os.Stat(cleanupPath); err != nil {
		t.Fatalf("cleanup manifest not created at %s: %v", cleanupPath, err)
	}

	// Verify installation manifest updated
	manifest, err := install.LoadManifest(home)
	if err != nil {
		t.Fatalf("failed to load installation manifest: %v", err)
	}
	if manifest.Connectors["opencode"] != "installed" {
		t.Fatalf("expected opencode in connectors manifest, got %v", manifest.Connectors)
	}

	// Uninstall
	uninstRes, err := c.Uninstall(home, projectRoot, connectors.UninstallOptions{})
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if !uninstRes.Clean {
		t.Fatalf("expected clean uninstall, got leftovers: %v", uninstRes.Leftovers)
	}
	if len(uninstRes.Removed) == 0 {
		t.Fatal("expected removed paths to be non-empty")
	}

	// Verify installation manifest updated after uninstall
	manifestAfter, err := install.LoadManifest(home)
	if err != nil {
		t.Fatalf("failed to reload manifest: %v", err)
	}
	if _, exists := manifestAfter.Connectors["opencode"]; exists {
		t.Errorf("opencode still present in connectors manifest after uninstall")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && stringSearch(s, substr)))
}

func stringSearch(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
