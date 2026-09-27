package autoupdate

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		v1, v2 string
		want   int
	}{
		{"0.6.0", "0.6.0", 0},
		{"v0.6.0", "0.6.0", 0},
		{"0.6.0", "0.6.1", -1},
		{"0.6.1", "0.6.0", 1},
		{"0.5.9", "0.6.0", -1},
		{"0.7.0", "0.6.9", 1},
		{"1.0.0", "0.9.9", 1},
		{"0.6.0-rc1", "0.6.0", -1},
		{"0.6.0", "0.6.0-rc1", 1},
		{"0.6.0-rc1", "0.6.0-rc2", -1},
	}

	for _, tt := range tests {
		got := CompareSemver(tt.v1, tt.v2)
		if got != tt.want {
			t.Errorf("CompareSemver(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
		}
	}
}

func TestCheckUpdateAvailable(t *testing.T) {
	assetName := TargetAssetName(runtime.GOOS, runtime.GOARCH)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/test-org/prumo/releases/latest" {
			http.NotFound(w, r)
			return
		}
		rel := GitHubRelease{
			TagName:     "v0.7.0",
			Name:        "Prumo v0.7.0",
			PublishedAt: "2026-09-26T12:00:00Z",
			Assets: []ReleaseAsset{
				{Name: assetName, BrowserDownloadURL: "http://example.com/" + assetName, Size: 1024},
				{Name: "checksums.txt", BrowserDownloadURL: "http://example.com/checksums.txt", Size: 256},
			},
		}
		_ = json.NewEncoder(w).Encode(rel)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	opts := CheckOptions{
		Repository:     "test-org/prumo",
		CurrentVersion: "0.6.0",
		BaseURL:        server.URL,
		SkipCache:      true,
		CacheDir:       tempDir,
		HTTPClient:     server.Client(),
	}

	res, err := Check(context.Background(), opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if !res.HasUpdate {
		t.Errorf("expected HasUpdate to be true (current: 0.6.0, latest: 0.7.0)")
	}
	if res.LatestVersion != "0.7.0" {
		t.Errorf("expected LatestVersion = 0.7.0, got %s", res.LatestVersion)
	}
	if res.Release.AssetName != assetName {
		t.Errorf("expected AssetName = %s, got %s", assetName, res.Release.AssetName)
	}
}

func TestDownloadAndVerify_Success(t *testing.T) {
	assetName := TargetAssetName(runtime.GOOS, runtime.GOARCH)
	fakeBinaryContent := []byte("#!/bin/sh\necho prumo v0.7.0\n")
	hasher := sha256.New()
	hasher.Write(fakeBinaryContent)
	expectedSHA := fmt.Sprintf("%x", hasher.Sum(nil))

	checksumsContent := fmt.Sprintf("%s  %s\n", expectedSHA, assetName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/binary":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fakeBinaryContent)
		case "/checksums.txt":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(checksumsContent))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	rel := &ReleaseInfo{
		Version:      "0.7.0",
		TagName:      "v0.7.0",
		AssetName:    assetName,
		AssetURL:     server.URL + "/binary",
		ChecksumsURL: server.URL + "/checksums.txt",
	}

	tempDir := t.TempDir()
	binPath, sha, err := DownloadAndVerify(context.Background(), server.Client(), rel, tempDir)
	if err != nil {
		t.Fatalf("DownloadAndVerify failed: %v", err)
	}

	if sha != expectedSHA {
		t.Errorf("expected sha %s, got %s", expectedSHA, sha)
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("downloaded binary does not exist at %s: %v", binPath, err)
	}
}

func TestDownloadAndVerify_TamperedBinaryRejected(t *testing.T) {
	assetName := TargetAssetName(runtime.GOOS, runtime.GOARCH)
	fakeBinaryContent := []byte("tampered binary payload")
	wrongSHA := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	checksumsContent := fmt.Sprintf("%s  %s\n", wrongSHA, assetName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/binary":
			_, _ = w.Write(fakeBinaryContent)
		case "/checksums.txt":
			_, _ = w.Write([]byte(checksumsContent))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	rel := &ReleaseInfo{
		Version:      "0.7.0",
		TagName:      "v0.7.0",
		AssetName:    assetName,
		AssetURL:     server.URL + "/binary",
		ChecksumsURL: server.URL + "/checksums.txt",
	}

	tempDir := t.TempDir()
	_, _, err := DownloadAndVerify(context.Background(), server.Client(), rel, tempDir)
	if err == nil {
		t.Fatal("expected failure due to supply chain checksum mismatch, but got nil")
	}
	if !strings.Contains(err.Error(), "supply chain verification failure") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestApplyAndRollback(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "bin", "prumo")
	_ = os.MkdirAll(filepath.Dir(targetPath), 0755)

	// Create initial binary
	if err := os.WriteFile(targetPath, []byte("v0.6.0"), 0755); err != nil {
		t.Fatalf("failed to create initial target: %v", err)
	}

	// Create new downloaded binary
	downloadedPath := filepath.Join(tempDir, "prumo-new")
	if err := os.WriteFile(downloadedPath, []byte("v0.7.0"), 0755); err != nil {
		t.Fatalf("failed to create downloaded binary: %v", err)
	}

	// Apply
	backupPath, err := Apply(downloadedPath, targetPath)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	newContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read target: %v", err)
	}
	if string(newContent) != "v0.7.0" {
		t.Errorf("target content = %s, want v0.7.0", string(newContent))
	}

	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}
	if string(backupContent) != "v0.6.0" {
		t.Errorf("backup content = %s, want v0.6.0", string(backupContent))
	}

	// Test Rollback
	if err := Rollback(targetPath, backupPath); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	rolledBackContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read rolled back target: %v", err)
	}
	if string(rolledBackContent) != "v0.6.0" {
		t.Errorf("rolled back content = %s, want v0.6.0", string(rolledBackContent))
	}
}
