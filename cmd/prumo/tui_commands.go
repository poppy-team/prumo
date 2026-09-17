package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	prumo "github.com/raillen/prumo/sdk/prumo"
	"github.com/raillen/prumo/tui"
)

// runTui is `prumo tui`: the terminal client.
//
// This is the only place where the TUI and the daemon are wired together, and
// it wires them the way any third-party client would: start the daemon as a
// subprocess, then talk to it over the Agent Protocol. Nothing here reaches
// into the daemon's internals, which is what lets `tui/` stay free of them.
func runTui(asJSON bool, args []string) int {
	f := agentFlags(args)
	if asJSON {
		// An interactive surface has no envelope. Refusing is better than
		// printing JSON and then taking over the terminal.
		return serviceError(true, fmt.Errorf("tui is interactive and has no --json output"))
	}
	workspace := f["path"]
	if workspace == "" {
		workspace = "."
	}
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return serviceError(asJSON, err)
	}

	storeDir := filepath.Join(abs, ".prumo", "runtime", "harness")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// A named --socket means the caller already has a daemon; starting a
	// second one would fight it for the store lock. A --remote endpoint means
	// the daemon is elsewhere for the same reason.
	var supervised *tui.Daemon
	if f["socket"] == "" && f["remote"] == "" {
		supervised = tui.NewDaemon(abs)
		if err := supervised.Start(ctx, 15*time.Second); err != nil {
			return serviceError(asJSON, err)
		}
		defer supervised.Stop()
	}

	client, err := tuiClient(f, tui.SocketPathFor(abs))
	if err != nil {
		return serviceError(asJSON, err)
	}
	if err := waitForDaemon(ctx, client, 5*time.Second); err != nil {
		return serviceError(asJSON, err)
	}

	styles, err := tui.NewStyles(f["theme"])
	if err != nil {
		return serviceError(asJSON, err)
	}
	model := tui.NewModel(styles, abs, storeDir)
	model.Config.Provider = defaultString(f["provider"], "fake")
	model.Config.Model = f["model"]
	model.Config.MaxTurns = positiveInt(f["max-turns"], model.Config.MaxTurns)
	model.SetSessionFactory(func(ctx context.Context, cfg tui.StartConfig) (*tui.Session, error) {
		return tui.StartSession(ctx, client, cfg)
	})

	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		return serviceError(asJSON, err)
	}
	return exitOK
}

// tuiClient maps the TUI's flags onto an SDK client.
//
// A remote endpoint is the same client at another address rather than a second
// code path — that is the whole of H10 criterion 4, and it only holds because
// the TUI talks to the daemon exclusively through the public SDK.
func tuiClient(f map[string]string, workspaceSocket string) (prumo.Client, error) {
	addr := f["remote"]
	if addr == "" {
		if socket := f["socket"]; socket != "" {
			return prumo.Client{SocketPath: socket}, nil
		}
		return prumo.Client{SocketPath: workspaceSocket}, nil
	}
	token := f["token"]
	if token == "" {
		if file := f["token-file"]; file != "" {
			data, err := os.ReadFile(file)
			if err != nil {
				return prumo.Client{}, fmt.Errorf("token file: %w", err)
			}
			token = strings.TrimSpace(string(data))
		}
	}
	if token == "" {
		token = os.Getenv("PRUMO_DAEMON_TOKEN")
	}
	if token == "" {
		return prumo.Client{}, fmt.Errorf("--remote %s requires a token (--token/--token-file/PRUMO_DAEMON_TOKEN)", addr)
	}
	conf := &tls.Config{MinVersion: tls.VersionTLS12}
	if ca := f["remote-tls-cert"]; ca != "" {
		pem, err := os.ReadFile(ca)
		if err != nil {
			return prumo.Client{}, fmt.Errorf("ca cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return prumo.Client{}, fmt.Errorf("ca cert: no certificates parsed")
		}
		conf.RootCAs = pool
	}
	return *prumo.DialRemote(addr, token, conf), nil
}

// waitForDaemon confirms the endpoint answers before the TUI takes the screen.
// Discovering an unreachable daemon *after* entering the alternate screen would
// leave the user staring at an empty view.
func waitForDaemon(ctx context.Context, client prumo.Client, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		probe, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		_, err := client.Protocol(probe)
		cancel()
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("no harness daemon answered: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func positiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
