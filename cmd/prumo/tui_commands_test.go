package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTuiClientMapsFlags pins the flag → client mapping. Criterion 4 is "the
// same client at another address", and that only holds if the address and its
// credentials actually reach the SDK client.
func TestTuiClientMapsFlags(t *testing.T) {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token")
	if err := os.WriteFile(tokenFile, []byte("  tok-remote\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	local, err := tuiClient(map[string]string{}, "/ws/agentd.sock")
	if err != nil {
		t.Fatalf("default client: %v", err)
	}
	if local.RemoteAddr != "" || local.SocketPath != "/ws/agentd.sock" {
		t.Fatalf("default client is not the workspace socket: %+v", local)
	}

	named, err := tuiClient(map[string]string{"socket": "/tmp/named.sock"}, "/ws/agentd.sock")
	if err != nil {
		t.Fatalf("named socket: %v", err)
	}
	if named.SocketPath != "/tmp/named.sock" {
		t.Fatalf("--socket was ignored: %+v", named)
	}

	if _, err := tuiClient(map[string]string{"remote": "127.0.0.1:7777"}, "/ws/agentd.sock"); err == nil {
		t.Fatal("a remote endpoint without a token must be refused")
	}

	remote, err := tuiClient(map[string]string{"remote": "127.0.0.1:7777", "token-file": tokenFile}, "/ws/agentd.sock")
	if err != nil {
		t.Fatalf("remote client: %v", err)
	}
	if remote.RemoteAddr != "127.0.0.1:7777" || remote.RemoteToken != "tok-remote" {
		t.Fatalf("remote mapping lost the address or token: %+v", remote)
	}
	if remote.TLSConf == nil || remote.TLSConf.MinVersion == 0 {
		t.Fatalf("remote client must carry a TLS configuration: %+v", remote.TLSConf)
	}
	if remote.SocketPath != "" {
		t.Fatalf("a remote client must not also hold a socket: %+v", remote)
	}

	badCA := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(badCA, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tuiClient(map[string]string{"remote": "127.0.0.1:7777", "token": "t", "remote-tls-cert": badCA}, "/ws/agentd.sock"); err == nil {
		t.Fatal("an unparsable CA must be refused, not silently ignored")
	}
}
