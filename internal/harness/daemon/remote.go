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
	"sync"
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
				// Same as Serve: closing the listener is not the end of the
				// daemon's job, and a run in flight has a record to write
				// (GAP-105).
				s.drain()
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
	// The request loop and the subscription goroutine both write to this
	// connection. The local handler serialises them; a remote one that did not
	// would interleave two writers into one bufio.Writer (GAP-166).
	var writeMu sync.Mutex
	safeWrite := func(v map[string]any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writeMsg(w, v)
	}
	authed := false
	var stop replacedSubscription
	// The stream outlives the request loop, so it needs a context of its own
	// that a closed connection cancels.
	remoteCtx, cancelRemote := context.WithCancel(s.rootCtx)
	defer cancelRemote()
	defer func() {
		if stop != nil {
			stop()
		}
	}()
	for sc.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			_ = safeWrite(map[string]any{"ok": false, "error": "invalid json"})
			continue
		}
		if !authed {
			got, _ := msg["auth"].(string)
			if subtleCompare(got, token) {
				authed = true
				_ = safeWrite(map[string]any{"ok": true, "authed": true})
				continue
			}
			_ = safeWrite(map[string]any{"ok": false, "error": "unauthorized"})
			return
		}
		if str(msg, "op") == "subscribe" {
			// A subscription is a stream, not a reply. Dispatching it answered
			// the request with one result and closed the connection, so a client
			// that subscribed over TCP got a single event and then silence
			// (GAP-166).
			stop.replace(s, remoteCtx, msg, safeWrite)
			continue
		}
		if err := safeWrite(s.dispatch(msg)); err != nil {
			return
		}
	}
}

// subtleCompare avoids early-exit timing leaks on tokens.
func subtleCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
