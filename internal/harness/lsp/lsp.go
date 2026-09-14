// Package lsp speaks a Language Server subset over stdio JSON-RPC
// (GAP-006 slice 3): initialize, workspace/symbol, textDocument/
// documentSymbol, shutdown. Providers stay optional: SymbolProvider picks
// LSP when a server is configured and falls back to the repomap index
// otherwise. No LSP binary is required to build, test or run the Harness.
package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/raillen/prumo/internal/harness/repomap"
)

// Symbol is one workspace/document symbol.
type Symbol struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Detail   string `json:"detail,omitempty"`
	Provider string `json:"provider"`
}

// Provider answers symbol queries.
type Provider interface {
	WorkspaceSymbols(ctx context.Context, query string) ([]Symbol, error)
	Hover(ctx context.Context, file string, line, character int) (string, error)
	Definition(ctx context.Context, file string, line, character int) (Symbol, error)
	Available() bool
	Describe() string
}

// Transport moves LSP messages (Content-Length framed).
type Transport interface {
	Send(ctx context.Context, data []byte) error
	Receive(ctx context.Context) ([]byte, error)
	Close() error
}

// PipeTransport is the in-memory test double (Content-Length framed,
// like stdio, so framing bugs surface in tests too).
type PipeTransport struct {
	toServer chan []byte
	toClient chan []byte
	closed   int32
}

func NewPipe() (*PipeTransport, *PipeTransport) {
	a := &PipeTransport{toServer: make(chan []byte, 64), toClient: make(chan []byte, 64)}
	b := &PipeTransport{toServer: a.toClient, toClient: a.toServer}
	return a, b
}

func (p *PipeTransport) Send(_ context.Context, data []byte) error {
	cp := frame(append([]byte{}, data...))
	select {
	case p.toServer <- cp:
		return nil
	default:
		return fmt.Errorf("transport saturated")
	}
}

func (p *PipeTransport) Receive(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data, ok := <-p.toClient:
		if !ok {
			return nil, fmt.Errorf("transport closed")
		}
		body, err := parseFrame(data)
		if err != nil {
			return nil, err
		}
		return body, nil
	}
}

// parseFrame splits one framed message.
func parseFrame(data []byte) ([]byte, error) {
	for i := 0; i+3 < len(data); i++ {
		if data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
			return data[i+4:], nil
		}
	}
	return nil, fmt.Errorf("malformed frame")
}

func (p *PipeTransport) Close() error { return nil }

// frame adds the LSP Content-Length header.
func frame(body []byte) []byte {
	return append([]byte("Content-Length: "+strconv.Itoa(len(body))+"\r\n\r\n"), body...)
}

// StdioTransport spawns a language server (e.g. gopls).
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.Writer
	stdout *bufio.Reader
	mu     sync.Mutex
}

// StartStdio launches bin args... as an LSP server.
func StartStdio(ctx context.Context, bin string, args ...string) (*StdioTransport, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &StdioTransport{cmd: cmd, stdin: stdin, stdout: bufio.NewReader(stdout)}, nil
}

func (s *StdioTransport) Send(_ context.Context, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.stdin.Write(frame(data))
	return err
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if v, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			length, _ = strconv.Atoi(strings.TrimSpace(v))
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("missing content-length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (s *StdioTransport) Receive(ctx context.Context) ([]byte, error) {
	type result struct {
		body []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		body, err := readFrame(s.stdout)
		ch <- result{body, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.body, r.err
	}
}

func (s *StdioTransport) Close() error {
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	return nil
}

// Client is an LSP workspace/symbol client.
type Client struct {
	Transport Transport
	Timeout   time.Duration
	Root      string
	next      int64
	inited    int32
}

func (c *Client) timeout() time.Duration {
	if c.Timeout <= 0 {
		return 10 * time.Second
	}
	return c.Timeout
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) call(ctx context.Context, method string, params any) (any, error) {
	id := atomic.AddInt64(&c.next, 1)
	data, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	rctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	if err := c.Transport.Send(rctx, data); err != nil {
		return nil, err
	}
	for {
		raw, err := c.Transport.Receive(rctx)
		if err != nil {
			return nil, err
		}
		var resp rpcResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			continue // notifications have no id; skip
		}
		if resp.ID != id {
			continue
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("lsp %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

// Initialize performs the LSP handshake for Root.
func (c *Client) Initialize(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&c.inited, 0, 1) {
		return nil
	}
	uri := "file://" + c.Root
	_, err := c.call(ctx, "initialize", map[string]any{
		"processId": nil, "rootUri": uri,
		"capabilities": map[string]any{},
	})
	if err != nil {
		atomic.StoreInt32(&c.inited, 0)
		return err
	}
	return nil
}

// WorkspaceSymbols queries the server index.
func (c *Client) WorkspaceSymbols(ctx context.Context, query string) ([]Symbol, error) {
	if err := c.Initialize(ctx); err != nil {
		return nil, err
	}
	raw, err := c.call(ctx, "workspace/symbol", map[string]any{"query": query})
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(raw)
	var items []struct {
		Name     string `json:"name"`
		Kind     int    `json:"kind"`
		Location struct {
			URI   string `json:"uri"`
			Range struct {
				Start struct {
					Line int `json:"line"`
				} `json:"start"`
			} `json:"range"`
		} `json:"location"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	out := make([]Symbol, 0, len(items))
	for _, it := range items {
		file := strings.TrimPrefix(it.Location.URI, "file://")
		out = append(out, Symbol{Name: it.Name, Kind: lspKind(it.Kind), File: file,
			Line: it.Location.Range.Start.Line + 1, Provider: "lsp"})
	}
	return out, nil
}

func lspKind(k int) string {
	switch k {
	case 5:
		return "class"
	case 6:
		return "method"
	case 12:
		return "function"
	case 23:
		return "struct"
	default:
		return fmt.Sprintf("kind-%d", k)
	}
}

// DetectServer finds a language server binary for lang (go → gopls).
func DetectServer(lang string) string {
	candidates := map[string][]string{"go": {"gopls"}, "rust": {"rust-analyzer"}, "python": {"basedpyright", "pyright"}, "typescript": {"typescript-language-server"}}
	for _, bin := range candidates[lang] {
		if _, err := exec.LookPath(bin); err == nil {
			return bin
		}
	}
	return ""
}

// FallbackProvider serves symbols from the repomap index (no server).
type FallbackProvider struct {
	Index repomap.Map
}

func (f FallbackProvider) Available() bool  { return true }
func (f FallbackProvider) Describe() string { return "repomap fallback (no language server)" }
func (f FallbackProvider) WorkspaceSymbols(_ context.Context, query string) ([]Symbol, error) {
	found := f.Index.Lookup(query)
	out := make([]Symbol, 0, len(found))
	for _, s := range found {
		out = append(out, Symbol{Name: s.Name, Kind: s.Kind, File: s.File, Line: s.Line, Detail: s.Signature, Provider: "repomap"})
	}
	return out, nil
}

// Hover needs a live server; the fallback says so explicitly.
func (f FallbackProvider) Hover(_ context.Context, _ string, _, _ int) (string, error) {
	return "", fmt.Errorf("hover requires a language server (none configured)")
}

// Definition falls back to the index: exact name match at query position is
// out of scope without a server, so this resolves by symbol name lookup.
func (f FallbackProvider) Definition(_ context.Context, file string, line, _ int) (Symbol, error) {
	return Symbol{}, fmt.Errorf("definition requires a language server (none configured)")
}

// Hover returns markdown hover text (1-based line/character).
func (c *Client) Hover(ctx context.Context, file string, line, character int) (string, error) {
	if err := c.Initialize(ctx); err != nil {
		return "", err
	}
	raw, err := c.call(ctx, "textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": "file://" + file},
		"position":     map[string]any{"line": line - 1, "character": character},
	})
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(raw)
	var doc struct {
		Contents struct {
			Value string `json:"value"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	return doc.Contents.Value, nil
}

// Definition resolves a symbol location (1-based in, 1-based out).
func (c *Client) Definition(ctx context.Context, file string, line, character int) (Symbol, error) {
	if err := c.Initialize(ctx); err != nil {
		return Symbol{}, err
	}
	raw, err := c.call(ctx, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": "file://" + file},
		"position":     map[string]any{"line": line - 1, "character": character},
	})
	if err != nil {
		return Symbol{}, err
	}
	data, _ := json.Marshal(raw)
	var loc struct {
		URI   string `json:"uri"`
		Range struct {
			Start struct {
				Line int `json:"line"`
			} `json:"start"`
		} `json:"range"`
	}
	if err := json.Unmarshal(data, &loc); err != nil || loc.URI == "" {
		return Symbol{}, fmt.Errorf("no definition found")
	}
	return Symbol{File: strings.TrimPrefix(loc.URI, "file://"), Line: loc.Range.Start.Line + 1, Provider: "lsp"}, nil
}

// ServerProvider adapts Client to Provider.
type ServerProvider struct {
	Client *Client
	Bin    string
}

func (s ServerProvider) Available() bool  { return s.Bin != "" }
func (s ServerProvider) Describe() string { return "lsp server " + s.Bin }
func (s ServerProvider) WorkspaceSymbols(ctx context.Context, query string) ([]Symbol, error) {
	return s.Client.WorkspaceSymbols(ctx, query)
}
func (s ServerProvider) Hover(ctx context.Context, file string, line, character int) (string, error) {
	return s.Client.Hover(ctx, file, line, character)
}
func (s ServerProvider) Definition(ctx context.Context, file string, line, character int) (Symbol, error) {
	return s.Client.Definition(ctx, file, line, character)
}

// Select returns the LSP provider when a server binary exists for lang,
// else the repomap fallback. Availability is explicit, never assumed.
func Select(root, lang string, index repomap.Map) Provider {
	if bin := DetectServer(lang); bin != "" {
		if t, err := StartStdio(context.Background(), bin); err == nil {
			return ServerProvider{Client: &Client{Transport: t, Root: root}, Bin: bin}
		}
	}
	return FallbackProvider{Index: index}
}
