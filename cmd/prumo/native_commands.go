// Command wiring for `prumo native` (and `prumo viewer`).
//
// The native desktop Workspace Viewer is not compiled into this Go binary.
// It lives in its own Rust crate (`prumo-viewer`, producing `prumo-native`)
// and communicates with the Go harness strictly over IPC (Unix domain socket
// or TCP), preserving the core boundary and the invariant `Run != GUI`.
// This launcher resolves the viewer binary and forwards CLI flags.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/protocol"
)

func runNative(asJSON bool, args []string) int {
	if asJSON {
		return serviceError(true, fmt.Errorf("native viewer is an interactive desktop GUI and has no --json output"))
	}

	binary, cargoArgs, err := resolveNativeBinary()
	if err != nil {
		return serviceError(asJSON, err)
	}

	forwarded := append([]string{}, cargoArgs...)
	forwarded = append(forwarded, args...)

	// Default workspace to current directory if not specified
	hasWorkspace := false
	for _, arg := range args {
		if arg == "--workspace" || arg == "-w" {
			hasWorkspace = true
			break
		}
	}
	if !hasWorkspace && len(cargoArgs) == 0 {
		forwarded = append(forwarded, "--workspace", ".")
	}

	cmd := exec.Command(binary, forwarded...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return serviceError(asJSON, fmt.Errorf("prumo native viewer: %w", err))
	}
	return exitOK
}

func resolveNativeBinary() (string, []string, error) {
	// 1. Explicit environment variable
	if fromEnv := strings.TrimSpace(os.Getenv("PRUMO_NATIVE_BIN")); fromEnv != "" {
		return fromEnv, nil, nil
	}
	if fromEnv := strings.TrimSpace(os.Getenv("PRUMO_VIEWER_BIN")); fromEnv != "" {
		return fromEnv, nil, nil
	}

	// 2. Sibling binary in executable directory
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, name := range []string{"prumo-native", "prumo-viewer"} {
			sibling := filepath.Join(exeDir, name)
			if _, err := os.Stat(sibling); err == nil {
				return sibling, nil, nil
			}
		}
	}

	// 3. Local build target in repository
	root := repoRoot()
	if root != "" {
		targetRelease := filepath.Join(root, "prumo-viewer", "target", "release", "prumo-native")
		if _, err := os.Stat(targetRelease); err == nil {
			return targetRelease, nil, nil
		}
		targetDebug := filepath.Join(root, "prumo-viewer", "target", "debug", "prumo-native")
		if _, err := os.Stat(targetDebug); err == nil {
			return targetDebug, nil, nil
		}

		// If cargo is available and source manifest exists, fall back to cargo run
		cargoToml := filepath.Join(root, "prumo-viewer", "Cargo.toml")
		if _, err := os.Stat(cargoToml); err == nil {
			if cargoPath, err := exec.LookPath("cargo"); err == nil {
				return cargoPath, []string{"run", "--manifest-path", cargoToml, "--"}, nil
			}
		}
	}

	// 4. System PATH lookup
	for _, name := range []string{"prumo-native", "prumo-viewer"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil, nil
		}
	}

	return "", nil, fmt.Errorf("the native workspace viewer is a separate binary: build `prumo-viewer` (producing `prumo-native`), or set PRUMO_NATIVE_BIN (protocol %s)", protocol.CLIVersion)
}
