// Command wiring for `prumo tui`.
//
// The terminal client is not part of this binary. It lives in its own module
// (`prumo-tui`) and speaks to the harness only through the public SDK, which is
// what makes the client/core boundary structural rather than a matter of
// discipline. This file is therefore a launcher: it resolves the client binary
// and hands it the flags it understands.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/protocol"
)

func runTui(asJSON bool, args []string) int {
	if asJSON {
		// An interactive surface has no envelope. Refusing is better than
		// printing JSON and then taking over the terminal.
		return serviceError(true, fmt.Errorf("tui is interactive and has no --json output"))
	}
	binary, err := tuiBinary()
	if err != nil {
		return serviceError(asJSON, err)
	}
	// Flags are forwarded verbatim: the two binaries must not drift into
	// disagreeing about what one means.
	forwarded := append([]string{}, args...)
	if _, ok := agentFlags(args)["path"]; !ok {
		forwarded = append(forwarded, "--path", ".")
	}

	cmd := exec.Command(binary, forwarded...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return serviceError(asJSON, fmt.Errorf("prumo agent: %w", err))
	}
	return exitOK
}

// tuiBinary resolves the client executable.
//
// PRUMO_AGENT_BIN / PRUMO_TUI_BIN wins so a second build can be named explicitly;
// otherwise sibling `pa` (or `prumo-agent`) is used; PATH lookup is the last resort.
func tuiBinary() (string, error) {
	if fromEnv := strings.TrimSpace(os.Getenv("PRUMO_AGENT_BIN")); fromEnv != "" {
		return fromEnv, nil
	}
	if fromEnv := strings.TrimSpace(os.Getenv("PRUMO_TUI_BIN")); fromEnv != "" {
		return fromEnv, nil
	}
	for _, name := range []string{"pa", "prumo-agent", "prumo-agent-tui"} {
		if exe, err := os.Executable(); err == nil {
			sibling := filepath.Join(filepath.Dir(exe), name)
			if _, err := os.Stat(sibling); err == nil {
				return sibling, nil
			}
		}
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("the terminal client is a separate binary: install `pa` (or `prumo-agent`), or set PRUMO_AGENT_BIN (protocol %s)", protocol.CLIVersion)
}
