package tui

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	prumo "github.com/raillen/prumo/sdk/prumo"
	"github.com/raillen/prumo/tui/theme"
)

// selfSignedCert writes a throwaway server certificate. The TUI cannot import
// internal/harness/daemon (criterion 1 forbids it), so the fixture lives here
// rather than being shared with the daemon's own tests.
func selfSignedCert(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "prumo-tui-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatal(err)
	}
	_ = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	certOut.Close()
	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	_ = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	keyOut.Close()
	return certFile, keyFile
}

// freePort reserves then releases a port. The window between the two is a race
// the test accepts: it needs an address the daemon can bind, not a lease.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func startRemoteDaemon(t *testing.T, bin, workspace string, extra ...string) (addr, token string) {
	t.Helper()
	dir := t.TempDir()
	certFile, keyFile := selfSignedCert(t, dir)
	port := freePort(t)
	addr = fmt.Sprintf("127.0.0.1:%d", port)
	token = "s3cret-token"
	args := append([]string{
		"agent", "serve", "--path", workspace,
		"--listen", addr, "--tls-cert", certFile, "--tls-key", keyFile, "--token", token,
	}, extra...)
	cmd := exec.Command(bin, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting the remote daemon: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	return addr, token
}

// TestLiveRemoteDaemonFlow is H10 acceptance criterion 4, with criterion 2's
// whole flow on top: the same model, the same client, another address — over
// TCP with TLS and a token rather than a Unix socket.
//
// The certificate is self-signed and the suite trusts it explicitly. Verifying
// a real CA chain is the daemon's own test; what this one has to prove is that
// the *client path* is the same one.
func TestLiveRemoteDaemonFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("live daemon test skipped in -short mode")
	}
	bin := buildPrumo(t)
	workspace := t.TempDir()
	// --permission ask is what makes the approval step reachable: without a
	// policy that asks, a run never stops at the gate.
	addr, token := startRemoteDaemon(t, bin, workspace, "--permission", "ask")

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := prumo.DialRemote(addr, token, &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}) // test-only: self-signed fixture
	deadline := time.Now().Add(30 * time.Second)
	for {
		probe, stop := context.WithTimeout(ctx, 500*time.Millisecond)
		_, err := client.Protocol(probe)
		stop()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("remote daemon never answered: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := client.Protocol(ctx); err != nil {
		t.Fatalf("remote Protocol: %v", err)
	}
	// A wrong token must be refused over TLS just as it is over the socket.
	bad := prumo.DialRemote(addr, "wrong", &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12})
	if _, err := bad.Protocol(ctx); err == nil {
		t.Fatal("a wrong token must not authenticate against the remote daemon")
	}

	styles, err := NewStyles(theme.DefaultTheme)
	if err != nil {
		t.Fatalf("styles: %v", err)
	}
	model := NewModel(styles, workspace, filepath.Join(workspace, ".prumo", "runtime", "harness"))
	model.Config.Provider = "fake-tools"
	model.Config.MaxTurns = 2
	model.SetSessionFactory(func(ctx context.Context, cfg StartConfig) (*Session, error) {
		return StartSession(ctx, client, cfg)
	})

	// palette → goal → run
	model = typeString(t, model, "run goal")
	model, _ = press(t, model, "enter")
	model = typeString(t, model, "prove the remote flow")
	model, cmd := press(t, model, "enter")
	if cmd == nil {
		t.Fatal("starting the run produced no command")
	}
	message := cmd()
	if _, ok := message.(sessionMsg); !ok {
		t.Fatalf("the remote daemon refused the run: %v", message)
	}
	model = update(t, model, message)
	if model.Stage() != StageRun {
		t.Fatalf("stage after start = %q, want the run panel", model.Stage())
	}

	// stream until the run stops for approval
	deadline = time.Now().Add(60 * time.Second)
	for {
		result, err := model.Session.Poll(ctx)
		if err != nil {
			t.Fatalf("poll: %v", err)
		}
		model = update(t, model, pollMsg{result: result})
		if result.Finished {
			t.Fatalf("the run ended without asking for approval: %+v", model.Session.Status())
		}
		if _, waiting := model.Session.PendingPermission(); waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the run never asked for approval; last = %+v", model.Session.Status())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if model.Stage() != StageRun {
		t.Fatalf("stage while awaiting approval = %q, want the run panel", model.Stage())
	}
	if rendered := model.View().Content; !strings.Contains(rendered, "a to approve") {
		t.Errorf("the run panel does not offer the decision:\n%s", rendered)
	}

	// approve over the remote transport, then stream to the end
	model, cmd = press(t, model, "a")
	if cmd == nil {
		t.Fatal("approving produced no command")
	}
	model = update(t, model, cmd())
	deadline = time.Now().Add(60 * time.Second)
	for {
		result, err := model.Session.Poll(ctx)
		if err != nil {
			t.Fatalf("poll: %v", err)
		}
		model = update(t, model, pollMsg{result: result})
		if result.Finished {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the run never finished after approval; last = %+v", model.Session.Status())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if model.Stage() != StageEvidence {
		t.Fatalf("stage after the run stopped = %q, want the evidence panel", model.Stage())
	}
	evidence := model.Session.Evidence(model.Timeline.Len(), model.Timeline.Dropped())
	if evidence.Status != "complete" {
		t.Fatalf("remote run status = %q (stop reason %q), want complete", evidence.Status, evidence.StopReason)
	}
	if !strings.Contains(model.View().Content, evidence.RunID) {
		t.Errorf("evidence panel does not name the run:\n%s", model.View().Content)
	}
}
