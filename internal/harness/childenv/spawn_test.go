package childenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The unit tests prove the rule. This proves the rule is reached: a real child
// process, spawned the way the MCP, ACP and opencode spawners spawn one, must
// not be able to read a secret out of its own environment.

func TestARealChildCannotReadTheHarnessSecret(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the helper is a shell script")
	}
	dir := t.TempDir()
	helper := filepath.Join(dir, "dump-env.sh")
	script := "#!/bin/sh\nenv | sort\n"
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	// A secret in the parent's environment, of a name the pass-through rules
	// have never heard of.
	const secretName = "A_VENDOR_WE_HAVE_NEVER_SEEN"
	const secretValue = "super-secret-token-value"
	t.Setenv(secretName, secretValue)
	t.Setenv("PRUMO_MODEL_API_KEY", "sk-harness-credential")

	cmd := exec.Command(helper)
	cmd.Env = Build(Options{})
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper failed: %v\n%s", err, out)
	}

	got := string(out)
	if strings.Contains(got, secretValue) {
		t.Fatalf("a child read the parent's secret from its own environment:\n%s", got)
	}
	if strings.Contains(got, "sk-harness-credential") {
		t.Fatalf("a child read the harness credential from its own environment:\n%s", got)
	}
	if !strings.Contains(got, "PATH=") {
		t.Fatalf("the child must still get PATH, or it cannot run anything:\n%s", got)
	}
}

func TestARealChildGetsAnExplicitlySetVariable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the helper is a shell script")
	}
	dir := t.TempDir()
	helper := filepath.Join(dir, "dump-env.sh")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nenv | sort\n"), 0o755); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	cmd := exec.Command(helper)
	cmd.Env = Build(Options{Extra: map[string]string{"PRUMO_CHILD_MARKER": "present"}})
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "PRUMO_CHILD_MARKER=present") {
		t.Fatalf("an explicitly set variable must reach the child:\n%s", out)
	}
}
