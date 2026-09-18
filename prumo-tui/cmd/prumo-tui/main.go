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

	tea "charm.land/bubbletea/v2"

	prumo "github.com/raillen/prumo/sdk/prumo"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/headless"
	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/stream"
	"github.com/raillen/prumo-tui/internal/tui"
	"github.com/raillen/prumo-tui/internal/tui/styles"
)

func main() {
	var (
		workspace     = flag.String("path", ".", "workspace root")
		socket        = flag.String("socket", "", "attach to this daemon socket instead of starting one")
		remote        = flag.String("remote", "", "attach to a daemon over TCP+TLS (requires --token)")
		token         = flag.String("token", "", "remote token")
		tokenFile     = flag.String("token-file", "", "file holding the remote token")
		tlsCert       = flag.String("remote-tls-cert", "", "CA certificate pinning the remote server")
		theme         = flag.String("theme", "", "theme id")
		provider      = flag.String("provider", "fake", "provider: fake|fake-tools|openai-compat|anthropic")
		model         = flag.String("model", "", "model id asked of the harness")
		maxTurns      = flag.Int("max-turns", 5, "maximum turns per run")
		reducedMotion = flag.Bool("reduced-motion", false, "state progress in words instead of animating it (also PRUMO_REDUCED_MOTION)")
		prompt        = flag.String("prompt", "", "run this goal and print what happened, without the interface")
		plain         = flag.Bool("plain", false, "with --prompt: print the run as prose, one fact per line")
		asJSON        = flag.Bool("json", false, "with --prompt: print the run as one JSON object per fact")
		version       = flag.Bool("version", false, "print the client version and exit")
	)
	flag.Parse()

	if *version {
		fmt.Println("prumo-agent-tui", Version)
		return
	}

	// Which glyphs the terminal can render is decided before anything is drawn,
	// since a replacement character where a label was meant cannot be undone.
	styles.ResolveIcons()

	abs, err := filepath.Abs(*workspace)
	if err != nil {
		fail(err)
	}

	// What the client wrote down for itself — its palette, so far — is read
	// before the flags, so an explicit flag still wins over the last session.
	if err := config.Load(); err != nil {
		fmt.Fprintln(os.Stderr, "prumo-tui: ignoring the client's own settings:", err)
	}

	cfg := *config.Get()
	if *theme != "" {
		cfg.Theme = *theme
	}
	cfg.WorkingDir = abs
	cfg.Provider = *provider
	cfg.Model = *model
	cfg.MaxTurns = *maxTurns
	// Reduced motion is a preference a user sets once, so the environment is
	// enough to hold it: a flag alone would have to be repeated on every launch.
	cfg.ReducedMotion = *reducedMotion || envEnabled("PRUMO_REDUCED_MOTION")
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

	// The signal context is cancelled on interrupt; the program itself is what
	// consumes the interrupt. It also bounds the event bridge, which has to stop
	// carrying events once the client is closing.
	streamCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
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

	// A goal given on the command line is not an interactive session: there is
	// nobody at the keyboard to answer a gate or read a repainting frame, so the
	// run is printed instead of drawn.
	if *prompt != "" {
		if err := runHeadless(streamCtx, *prompt, application, *plain, *asJSON); err != nil {
			fail(err)
		}
		return
	}

	program := tea.NewProgram(tui.New(application))
	// The bridge is what turns the services' events into messages the model
	// receives. Without it the program runs, draws and never learns that a run
	// produced anything.
	stream.Start(streamCtx, application, program)
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
	client := prumo.Client{SocketPath: sock}

	// A daemon left by an earlier run answers on this socket, and starting a
	// second one only makes it print "daemon already running" at a user who did
	// nothing wrong. Asking first is also cheaper than launching a process to
	// find out.
	if err := waitForDaemon(&client, 300*time.Millisecond); err == nil {
		return client, nil
	}

	stop, err := startDaemon(workspace, sock)
	if err != nil {
		return prumo.Client{}, err
	}
	// The daemon outlives this call by design; its lifetime is the client's.
	_ = stop
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
		sibling := filepath.Join(filepath.Dir(exe), "prumo-agent")
		if _, err := os.Stat(sibling); err == nil {
			return sibling, nil
		}
	}
	if path, err := exec.LookPath("prumo-agent"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("no prumo-agent binary found: set PRUMO_BIN or put `prumo-agent` on PATH")
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

// runHeadless prints a run instead of drawing it.
func runHeadless(ctx context.Context, goal string, application *app.App, plain, asJSON bool) error {
	mode := headless.ModePlain
	if asJSON {
		mode = headless.ModeJSON
	}
	_ = plain // plain is the default; the flag exists to say so out loud
	return headless.Run(ctx, application, headless.Options{Goal: goal, Mode: mode, Out: os.Stdout})
}

// envEnabled reports whether an environment variable asks for a preference.
//
// Unset, "0", "false" and "no" mean no; anything else means yes, so a user does
// not have to guess which spelling this client wanted.
func envEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "", "0", "false", "no":
		return false
	default:
		return true
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "prumo-tui:", err)
	os.Exit(1)
}
