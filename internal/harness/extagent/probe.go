// Provider availability probing: read-only capability discovery over the
// local host (binaries, versions, server reachability). Probes never start
// servers, spend tokens, or claim interop beyond what responds. Live matrix
// entries stay honest: present-but-unreachable is reported, not assumed.
package extagent

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// ProbeResult is one matrix row.
type ProbeResult struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"` // model | agent
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// Prober discovers provider availability. Zero value uses the process
// environment; tests override Path/Runner/Dial.
type Prober struct {
	Path        string // PATH override for binary lookup ("" = process PATH)
	Runner      func(ctx context.Context, bin string, args ...string) (string, error)
	Dial        func(ctx context.Context, addr string) error
	OpenCodeURL string // "" = http://127.0.0.1:4096
	ACPURL      string // "" = $PRUMO_ACP_URL, unset = unconfigured
}

func (p Prober) runner() func(context.Context, string, ...string) (string, error) {
	if p.Runner != nil {
		return p.Runner
	}
	return func(ctx context.Context, bin string, args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, bin, args...).CombinedOutput()
		return strings.TrimSpace(firstLine(string(out))), err
	}
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}

func (p Prober) lookPath(bin string) string {
	if p.Path != "" {
		for _, dir := range strings.Split(p.Path, string(os.PathListSeparator)) {
			full := dir + string(os.PathSeparator) + bin
			if st, err := os.Stat(full); err == nil && !st.IsDir() {
				return full
			}
		}
		return ""
	}
	found, _ := exec.LookPath(bin)
	return found
}

func (p Prober) dialer() func(context.Context, string) error {
	if p.Dial != nil {
		return p.Dial
	}
	return func(ctx context.Context, addr string) error {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
		if err != nil {
			return err
		}
		return conn.Close()
	}
}

// Matrix returns the availability table in stable order.
func (p Prober) Matrix(ctx context.Context) []ProbeResult {
	out := []ProbeResult{
		{Name: "fake", Kind: "model", Available: true, Version: "builtin", Detail: "deterministic conformance double"},
		p.openAICompat(),
		p.anthropic(),
		p.cliAgent("opencode-cli", "opencode", "--version"),
		p.openCodeServer(ctx),
		p.cliAgent("codex-cli", "codex", "--version"),
		p.acp(),
	}
	return out
}

// Sorted is Matrix ordered by name for golden comparisons.
func (p Prober) Sorted(ctx context.Context) []ProbeResult {
	out := p.Matrix(ctx)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func envVal(key string) string { return strings.TrimSpace(os.Getenv(key)) }

func (p Prober) openAICompat() ProbeResult {
	base := envVal("PRUMO_MODEL_BASE_URL")
	r := ProbeResult{Name: "openai-compat", Kind: "model"}
	if base == "" {
		r.Detail = "unconfigured: set PRUMO_MODEL_BASE_URL or --base-url"
		return r
	}
	r.Available = true
	r.Detail = "endpoint configured (keyed via PRUMO_MODEL_API_KEY)"
	return r
}

func (p Prober) anthropic() ProbeResult {
	r := ProbeResult{Name: "anthropic", Kind: "model"}
	if envVal("PRUMO_MODEL_API_KEY") == "" && envVal("PRUMO_MODEL_BASE_URL") == "" {
		r.Detail = "unconfigured: set PRUMO_MODEL_API_KEY (or PRUMO_MODEL_BASE_URL for a stub)"
		return r
	}
	r.Available = true
	r.Detail = "credentials/endpoint present (key never leaves the adapter)"
	return r
}

func (p Prober) cliAgent(name, bin, versionArg string) ProbeResult {
	r := ProbeResult{Name: name, Kind: "agent"}
	full := p.lookPath(bin)
	if full == "" {
		r.Detail = "binary " + bin + " not on PATH"
		return r
	}
	ver, err := p.runner()(context.Background(), full, versionArg)
	if err != nil {
		r.Available = true
		r.Detail = "present but --version failed: " + firstLine(err.Error())
		return r
	}
	r.Available = true
	r.Version = ver
	r.Detail = "binary " + full
	return r
}

func (p Prober) openCodeServer(ctx context.Context) ProbeResult {
	r := ProbeResult{Name: "opencode-server", Kind: "agent"}
	url := p.OpenCodeURL
	if url == "" {
		url = "http://127.0.0.1:4096"
	}
	host := strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://")
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}
	if err := p.dialer()(ctx, host); err != nil {
		r.Detail = "no server at " + url + " (start `opencode serve` or set --opencode-url)"
		return r
	}
	// Any HTTP answer (even 404) proves a live server; capability depth is
	// negotiated by the adapter, not assumed by the probe.
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, url+"/", nil)
	if resp, err := http.DefaultClient.Do(req); err == nil {
		resp.Body.Close()
		r.Version = resp.Status
	} else {
		r.Version = "tcp-open"
	}
	r.Available = true
	r.Detail = "reachable at " + url
	return r
}

func (p Prober) acp() ProbeResult {
	r := ProbeResult{Name: "acp-generic", Kind: "agent"}
	url := p.ACPURL
	if url == "" {
		url = envVal("PRUMO_ACP_URL")
	}
	if url == "" {
		r.Detail = "unconfigured: set PRUMO_ACP_URL to probe a generic ACP server"
		return r
	}
	r.Detail = "configured endpoint " + url + " (handshake on first use)"
	return r
}
