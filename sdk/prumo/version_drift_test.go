package prumo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	harnessprotocol "github.com/raillen/prumo/internal/harness/protocol"
)

// readRepoFile reads a file relative to the module root.
func readRepoFile(t *testing.T, rel string) (string, error) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
	data, err := os.ReadFile(filepath.Join(dir, rel))
	return string(data), err
}

// The SDK cannot import the engine's internals — a public client that compiles
// against internal packages is unusable outside the module — so it carries its
// own copy of the protocol version. A copy drifts. This pins the two together so
// the drift is a failing test rather than a client negotiating against a version
// nobody implements.
func TestSDKVersionMatchesEngine(t *testing.T) {
	if ProtocolVersion != harnessprotocol.Version {
		t.Fatalf("the SDK speaks %s but the engine speaks %s; update the SDK", ProtocolVersion, harnessprotocol.Version)
	}
}

// The TypeScript client is a second copy, in a second language, with no
// compiler to catch the difference.
func TestTypeScriptClientVersionMatchesEngine(t *testing.T) {
	source, err := readRepoFile(t, "sdk/typescript/client.ts")
	if err != nil {
		t.Fatal(err)
	}
	const marker = "export const PROTOCOL_VERSION ="
	idx := strings.Index(source, marker)
	if idx < 0 {
		t.Fatal("the TypeScript client does not declare PROTOCOL_VERSION")
	}
	rest := strings.TrimLeft(source[idx+len(marker):], " \t\r\n")
	quote := strings.IndexAny(rest, "\"'")
	if quote < 0 {
		t.Fatal("PROTOCOL_VERSION is not a quoted string")
	}
	inner := rest[quote+1:]
	end := strings.IndexAny(inner, "\"'")
	if end < 0 {
		t.Fatal("PROTOCOL_VERSION has an unterminated string")
	}
	declared := inner[:end]
	if declared != harnessprotocol.Version {
		t.Fatalf("the TypeScript client speaks %s but the engine speaks %s; update the client", declared, harnessprotocol.Version)
	}
}
