// Remote transport (GAP-030): optional TCP+TLS listener with token auth.
// Unix socket stays the default; remote is explicit (--listen) and always
// authenticated: clients present {"auth": token} as the first line, servers
// reject anything else before dispatch. Certificates are operator-provided;
// tests generate self-signed certs in-process.
package daemon

import (
	"bufio"
	"context"
	"crypto/subtle"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
)

// RemoteConfig enables the TCP+TLS listener. Empty ListenAddr disables it.
type RemoteConfig struct {
	ListenAddr string // e.g. 127.0.0.1:0 (port 0 = ephemeral, see Addr)
	CertFile   string
	KeyFile    string
	Token      string
}

// Addr returns the remote bound address after ServeRemote starts
// (useful with port 0); "" when remote is down.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rln == nil {
		return ""
	}
	return s.rln.Addr().String()
}

// ServeRemote serves the same dispatch over TLS with token auth.
func (s *Server) ServeRemote(ctx context.Context, cfg RemoteConfig) error {
	if cfg.ListenAddr == "" {
		return fmt.Errorf("remote listen address required")
	}
	if cfg.Token == "" {
		return fmt.Errorf("remote token required (refusing unauthenticated listeners)")
	}
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return fmt.Errorf("tls cert: %w", err)
	}
	ln, err := tls.Listen("tcp", cfg.ListenAddr, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.rln = ln
	s.mu.Unlock()
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				continue
			}
		}
		go s.handleRemote(conn, cfg.Token)
	}
}

func (s *Server) handleRemote(conn net.Conn, token string) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	w := bufio.NewWriter(conn)
	authed := false
	for sc.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			writeMsg(w, map[string]any{"ok": false, "error": "invalid json"})
			continue
		}
		if !authed {
			got, _ := msg["auth"].(string)
			if subtleCompare(got, token) {
				authed = true
				writeMsg(w, map[string]any{"ok": true, "authed": true})
				continue
			}
			writeMsg(w, map[string]any{"ok": false, "error": "unauthorized"})
			return
		}
		writeMsg(w, s.dispatch(msg))
	}
}

// subtleCompare avoids early-exit timing leaks on tokens.
func subtleCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
