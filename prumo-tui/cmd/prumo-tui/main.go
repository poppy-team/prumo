// Command prumo-tui is the Prumo terminal client.
//
// It is a client of the harness: it supervises (or attaches to) a
// `prumo agent serve` daemon and drives it over the agent protocol. It imports
// the public SDK and nothing from the core's internal tree — the module
// boundary is what makes that true rather than merely intended.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	prumo "github.com/raillen/prumo/sdk/prumo"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/tui"
)

func main() {
	var (
		workspace = flag.String("path", ".", "workspace root")
		socket    = flag.String("socket", "", "attach to this daemon socket instead of starting one")
		remote    = flag.String("remote", "", "attach to a daemon over TCP+TLS (requires --token)")
		token     = flag.String("token", "", "remote token")
		tokenFile = flag.String("token-file", "", "file holding the remote token")
		tlsCert   = flag.String("remote-tls-cert", "", "CA certificate pinning the remote server")
		theme     = flag.String("theme", "", "theme id")
		provider  = flag.String("provider", "fake", "provider: fake|fake-tools|openai-compat|anthropic")
		model     = flag.String("model", "", "model id asked of the harness")
		maxTurns  = flag.Int("max-turns", 5, "maximum turns per run")
		version   = flag.Bool("version", false, "print the client version and exit")
	)
	flag.Parse()

	if *version {
		fmt.Println("prumo-tui", Version)
		return
	}

	abs, err := filepath.Abs(*workspace)
	if err != nil {
		fail(err)
	}

	cfg := *config.Get()
	if *theme != "" {
		cfg.Theme = *theme
	}
	cfg.WorkingDir = abs
	cfg.Provider = *provider
	cfg.Model = *model
	cfg.MaxTurns = *maxTurns
	cfg.SocketPath = *socket
	cfg.RemoteAddr = *remote
	cfg.Token = *token
	cfg.TLSCert = *tlsCert
	if cfg.Token == "" && *tokenFile != "" {
		data, err := os.ReadFile(*tokenFile)
		if err != nil {
			fail(fmt.Errorf("token file: %w", err))
		}
		cfg.Token = strings.TrimSpace(string(data))
	}
	config.Set(cfg)

	// The signal context is cancelled on interrupt; the program itself is
	// what consumes the interrupt, so only the cleanup handle is kept.
	_, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	client, err := buildClient(cfg, abs)
	if err != nil {
		fail(err)
	}

	application := app.New(app.Options{
		Client:    &client,
		Provider:  cfg.Provider,
		Model:     models.Model{ID: models.ModelID(cfg.Model), Name: cfg.Model},
		MaxTurns:  cfg.MaxTurns,
		Workspace: abs,
	})

	program := tea.NewProgram(tui.New(application))
	if _, err := program.Run(); err != nil {
		fail(err)
	}
}

// buildClient returns a client for the configured endpoint, starting a daemon
// when no endpoint was given.
//
// A named socket or a remote address means somebody already owns the daemon;
// starting a second one would fight it for the store lock. Only the default
// case supervises one, which is what makes `prumo-tui` usable with no setup.
func buildClient(cfg config.Config, workspace string) (prumo.Client, error) {
	if cfg.RemoteAddr != "" {
		if cfg.Token == "" {
			return prumo.Client{}, fmt.Errorf("--remote %s requires a token (--token/--token-file/PRUMO_DAEMON_TOKEN)", cfg.RemoteAddr)
		}
		conf := &tls.Config{MinVersion: tls.VersionTLS12}
		if cfg.TLSCert != "" {
			pem, err := os.ReadFile(cfg.TLSCert)
			if err != nil {
				return prumo.Client{}, fmt.Errorf("ca cert: %w", err)
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(pem) {
				return prumo.Client{}, fmt.Errorf("ca cert: no certificates parsed")
			}
			conf.RootCAs = pool
		}
		client := prumo.DialRemote(cfg.RemoteAddr, cfg.Token, conf)
		if err := waitForDaemon(client, 10*time.Second); err != nil {
			return prumo.Client{}, err
		}
		return *client, nil
	}

	if cfg.SocketPath != "" {
		client := prumo.Client{SocketPath: cfg.SocketPath}
		if err := waitForDaemon(&client, 5*time.Second); err != nil {
			return prumo.Client{}, err
		}
		return client, nil
	}

	sock := filepath.Join(workspace, ".prumo", "runtime", "harness", "agentd.sock")
	stop, err := startDaemon(workspace, sock)
	if err != nil {
		return prumo.Client{}, err
	}
	// The daemon outlives this call by design; its lifetime is the client's.
	_ = stop
	client := prumo.Client{SocketPath: sock}
	if err := waitForDaemon(&client, 20*time.Second); err != nil {
		return prumo.Client{}, err
	}
	return client, nil
}

// startDaemon launches `prumo agent serve` and returns a stop function.
func startDaemon(workspace, socket string) (func(), error) {
	binary, err := prumoBinary()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o755); err != nil {
		return nil, fmt.Errorf("cannot create runtime dir: %w", err)
	}
	cmd := exec.Command(binary, "agent", "serve", "--path", workspace)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cannot start the daemon: %w", err)
	}
	return func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
		}
	}, nil
}

// prumoBinary resolves the daemon binary. PRUMO_BIN wins so a second build can
// be named explicitly; otherwise the sibling of this executable is used.
func prumoBinary() (string, error) {
	if fromEnv := os.Getenv("PRUMO_BIN"); fromEnv != "" {
		return fromEnv, nil
	}
	if exe, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(exe), "prumo")
		if _, err := os.Stat(sibling); err == nil {
			return sibling, nil
		}
	}
	if path, err := exec.LookPath("prumo"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("no prumo binary found: set PRUMO_BIN or put `prumo` on PATH")
}

// waitForDaemon calls the daemon rather than sleeping: readiness is the thing
// being asked about, so asking is the only honest way to decide it.
func waitForDaemon(client *prumo.Client, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last error
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, last = client.Protocol(ctx)
		cancel()
		if last == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("no harness daemon answered within %s: %w", timeout, last)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "prumo-tui:", err)
	os.Exit(1)
}
