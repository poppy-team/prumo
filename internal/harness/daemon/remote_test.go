package daemon

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdk "github.com/raillen/prumo/sdk/prumo"
)

func selfSigned(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "prumo-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  nil,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	certOut, _ := os.Create(certFile)
	_ = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	certOut.Close()
	keyOut, _ := os.Create(keyFile)
	_ = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	keyOut.Close()
	return certFile, keyFile
}

func TestRemoteAuthAndRoundtrip(t *testing.T) {
	dir := t.TempDir()
	srv := New("", filepath.Join(dir, "store"), fakeDeps(false))
	certFile, keyFile := selfSigned(t, dir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = srv.ServeRemote(ctx, RemoteConfig{ListenAddr: "127.0.0.1:0", CertFile: certFile, KeyFile: keyFile, Token: "s3cret"})
	}()
	deadline := time.Now().Add(5 * time.Second)
	for srv.Addr() == "" {
		if time.Now().After(deadline) {
			t.Fatal("remote listener did not come up")
		}
		time.Sleep(10 * time.Millisecond)
	}
	insecure := &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12} // test-only: cert pinned by suite
	bad := sdk.DialRemote(srv.Addr(), "wrong", insecure)
	if _, err := bad.Protocol(ctx); err == nil {
		t.Fatal("wrong token must be rejected")
	}
	c := sdk.DialRemote(srv.Addr(), "s3cret", insecure)
	info, err := c.Protocol(ctx)
	if err != nil || info.Version != sdk.ProtocolVersion {
		t.Fatalf("remote protocol failed: %+v %v", info, err)
	}
	id, err := c.Start(ctx, sdk.StartRequest{Goal: "remote run", Provider: "fake", RunID: "R-remote", MaxTurns: 1})
	if err != nil || id != "R-remote" {
		t.Fatalf("remote start failed: %q %v", id, err)
	}
	stCtx, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	st, err := c.Wait(stCtx, "R-remote", 20*time.Millisecond)
	if err != nil || st.Status != "complete" {
		t.Fatalf("remote wait failed: %+v %v", st, err)
	}
}

func TestRemoteRefusesConfigGaps(t *testing.T) {
	srv := New("", t.TempDir(), fakeDeps(false))
	ctx := context.Background()
	if err := srv.ServeRemote(ctx, RemoteConfig{}); err == nil {
		t.Fatal("missing addr must fail")
	}
	if err := srv.ServeRemote(ctx, RemoteConfig{ListenAddr: "127.0.0.1:0"}); err == nil {
		t.Fatal("missing token must fail")
	}
}
