package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	prumo "github.com/raillen/prumo/sdk/prumo"
)

// SocketPathFor is where the daemon listens for a given workspace. It must
// agree with `prumo agent serve`'s own default, because the TUI and the CLI are
// expected to find each other's daemon — a private path here would mean the TUI
// silently ran its own instance while `prumo agent ps` reported nothing.
func SocketPathFor(workspace string) string {
	return filepath.Join(workspace, ".prumo", "runtime", "harness", "agentd.sock")
}

// Daemon is a supervised `prumo agent serve` subprocess.
//
// The TUI supervises a real daemon rather than embedding one for two reasons
// that both pay off immediately: it keeps `tui/` free of `prumo/internal` (the
// boundary the H10 spike requires), and it exercises the *actual* protocol, so
// the remote mode in criterion 4 is the same client with another address instead
// of a second code path.
type Daemon struct {
	// Binary is the prumo executable to run. Empty means "the one this
	// process was started from".
	Binary string
	// Workspace is the repository root the daemon serves.
	Workspace string
	// Args are extra flags for `agent serve` (e.g. --permission ask). The TUI
	// does not set them itself: the run policy belongs to the daemon's
	// configuration, and a client that silently chose it would be deciding
	// something that is not its business.
	Args []string

	cmd  *exec.Cmd
	sock string
}

// NewDaemon returns a daemon supervisor for a workspace.
func NewDaemon(workspace string) *Daemon {
	return &Daemon{Workspace: workspace, sock: SocketPathFor(workspace)}
}

// Socket is the path the daemon listens on.
func (d *Daemon) Socket() string { return d.sock }

// Client returns an SDK client pointed at this daemon.
func (d *Daemon) Client() prumo.Client { return prumo.Client{SocketPath: d.sock} }

// binaryPath resolves the executable to run. PRUMO_BIN wins so a test (or a
// user with several builds) can name the binary explicitly; otherwise the
// running executable is used, which is the one that contains this code.
func (d *Daemon) binaryPath() (string, error) {
	if d.Binary != "" {
		return d.Binary, nil
	}
	if fromEnv := os.Getenv("PRUMO_BIN"); fromEnv != "" {
		return fromEnv, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("tui: cannot locate the prumo binary: %w", err)
	}
	return exe, nil
}

// Start launches the daemon and waits until it answers on the socket.
//
// Readiness is decided by *calling the daemon*, not by sleeping: the loop asks
// for the protocol description until it succeeds or the deadline passes. A
// fixed sleep would make the TUI flaky on slow machines and slow on fast ones.
func (d *Daemon) Start(ctx context.Context, readyTimeout time.Duration) error {
	if d.cmd != nil {
		return errors.New("tui: daemon already started")
	}
	binary, err := d.binaryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(d.sock), 0o755); err != nil {
		return fmt.Errorf("tui: cannot create runtime dir: %w", err)
	}
	args := append([]string{"agent", "serve", "--path", d.Workspace}, d.Args...)
	cmd := exec.Command(binary, args...)
	// The daemon's stdout is a human line ("serving harness daemon on …");
	// keeping it off the terminal is what lets the TUI own the screen.
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("tui: cannot start daemon: %w", err)
	}
	d.cmd = cmd

	if readyTimeout <= 0 {
		readyTimeout = 10 * time.Second
	}
	client := d.Client()
	deadline := time.Now().Add(readyTimeout)
	for {
		if err := ctx.Err(); err != nil {
			d.Stop()
			return err
		}
		probe, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		_, err := client.Protocol(probe)
		cancel()
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			d.Stop()
			return fmt.Errorf("tui: daemon did not become ready on %s within %s: %w", d.sock, readyTimeout, err)
		}
		select {
		case <-ctx.Done():
			d.Stop()
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// Stop terminates the daemon and waits for it, so no orphan process outlives
// the TUI. It is safe to call more than once.
func (d *Daemon) Stop() {
	if d.cmd == nil || d.cmd.Process == nil {
		return
	}
	_ = d.cmd.Process.Signal(os.Interrupt)
	done := make(chan struct{})
	go func() {
		_ = d.cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = d.cmd.Process.Kill()
		<-done
	}
	d.cmd = nil
}

// Running reports whether the daemon subprocess is still alive.
func (d *Daemon) Running() bool { return d.cmd != nil }
