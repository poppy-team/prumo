package prumo_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const tsCheckTpl = `import { Client } from %q;
const c = new Client(%q, 15000);
const id = await c.start({ goal: "ts e2e", provider: "fake", run_id: "R-ts", max_turns: 2 });
if (id !== "R-ts") throw new Error("bad id " + id);
const st = await c.wait("R-ts");
if (st.status !== "complete") throw new Error("bad status " + st.status);
const evs = await c.events("R-ts");
if (evs.length === 0) throw new Error("no events");
console.log("TS_E2E_PASS");
`

// TestTypeScriptClientRoundtrip proves the generated TS client against a
// live Go daemon. Skips without node or a buildable binary (hermetic CI).
func TestTypeScriptClientRoundtrip(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "prumo-ts")
	repo := repoRootOf(t)
	build := exec.Command("go", "build", "-o", bin, "./cmd/prumo")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Skipf("binary build unavailable: %s", out)
	}
	work := filepath.Join(dir, "work")
	sock := filepath.Join(dir, "agentd.sock")
	srv := exec.Command(bin, "agent", "serve", "--path", work, "--socket", sock)
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = exec.Command(bin, "agent", "stop", "--path", work).Run()
		_, _ = srv.Process.Wait()
	}()
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("daemon socket never appeared")
		}
		time.Sleep(100 * time.Millisecond)
	}
	script := filepath.Join(dir, "check.mts")
	scriptSrc := fmt.Sprintf(tsCheckTpl,
		filepath.Join(repo, "sdk", "typescript", "client.ts"), sock)
	if err := os.WriteFile(script, []byte(scriptSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, script)
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "TS_E2E_PASS") {
		t.Fatalf("ts roundtrip failed: %s %v", out, err)
	}
}

func repoRootOf(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
