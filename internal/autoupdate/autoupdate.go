// Package autoupdate implements secure, checksum-verified binary self-updates
// and release monitoring for the Prumo Harness and CLI directly from GitHub Releases.
package autoupdate

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/install"
	"github.com/raillen/prumo/internal/protocol"
)

const (
	DefaultRepository = "poppy-team/prumo"
	DefaultCacheTTL   = 4 * time.Hour
)

// ReleaseAsset describes one artifact attached to a GitHub release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// GitHubRelease represents the GitHub API payload for a release.
type GitHubRelease struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	PublishedAt string         `json:"published_at"`
	Prerelease  bool           `json:"prerelease"`
	Draft       bool           `json:"draft"`
	Assets      []ReleaseAsset `json:"assets"`
	HTMLURL     string         `json:"html_url"`
}

// ReleaseInfo holds the normalized release metadata needed for updates.
type ReleaseInfo struct {
	Version      string `json:"version"`
	TagName      string `json:"tag_name"`
	ReleaseNotes string `json:"release_notes,omitempty"`
	PublishedAt  string `json:"published_at,omitempty"`
	AssetURL     string `json:"asset_url"`
	AssetName    string `json:"asset_name"`
	ChecksumsURL string `json:"checksums_url"`
	Prerelease   bool   `json:"prerelease"`
	HTMLURL      string `json:"html_url,omitempty"`
}

// CheckOptions configures update discovery.
type CheckOptions struct {
	Repository     string
	CurrentVersion string
	Token          string
	BaseURL        string // Optional: for testing or GitHub Enterprise
	SkipCache      bool
	CacheDir       string
	HTTPClient     *http.Client
}

// CheckResult contains update status comparison.
type CheckResult struct {
	CurrentVersion string       `json:"current_version"`
	LatestVersion  string       `json:"latest_version"`
	HasUpdate      bool         `json:"has_update"`
	Release        *ReleaseInfo `json:"release,omitempty"`
	CheckedAt      string       `json:"checked_at"`
	FromCache      bool         `json:"from_cache,omitempty"`
}

// UpdateOptions configures the execution of an update.
type UpdateOptions struct {
	CheckOptions
	TargetVersion    string // If empty, updates to latest
	TargetBinaryPath string // If empty, detects via os.Executable() or installation manifest
	Force            bool
	DryRun           bool
}

// UpdateResult contains the status after an update execution.
type UpdateResult struct {
	PreviousVersion string `json:"previous_version"`
	UpdatedVersion  string `json:"updated_version"`
	BinaryPath      string `json:"binary_path"`
	BackupPath      string `json:"backup_path,omitempty"`
	VerifiedSHA256  string `json:"verified_sha256"`
	Success         bool   `json:"success"`
	Message         string `json:"message"`
}

type cachedCheck struct {
	Result    CheckResult `json:"result"`
	ExpiresAt time.Time   `json:"expires_at"`
}

// TargetAssetName returns the canonical asset name for the current OS and architecture.
func TargetAssetName(goos, goarch string) string {
	name := fmt.Sprintf("prumo-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// NormalizeVersion strips leading 'v' and whitespace for consistent semver parsing.
func NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	return strings.TrimPrefix(v, "v")
}

// CompareSemver compares two semver strings (e.g. "0.6.0" vs "0.6.1").
// Returns -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2.
func CompareSemver(v1, v2 string) int {
	clean1 := NormalizeVersion(v1)
	clean2 := NormalizeVersion(v2)

	// Separate pre-release tag if present (e.g. 0.6.0-rc1)
	pre1 := ""
	pre2 := ""
	if idx := strings.Index(clean1, "-"); idx >= 0 {
		pre1 = clean1[idx+1:]
		clean1 = clean1[:idx]
	}
	if idx := strings.Index(clean2, "-"); idx >= 0 {
		pre2 = clean2[idx+1:]
		clean2 = clean2[:idx]
	}

	parts1 := strings.Split(clean1, ".")
	parts2 := strings.Split(clean2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			n1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(parts2[i])
		}
		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	// If numeric parts are equal, versions without pre-release are higher than pre-release
	if pre1 == "" && pre2 != "" {
		return 1
	}
	if pre1 != "" && pre2 == "" {
		return -1
	}
	if pre1 < pre2 {
		return -1
	}
	if pre1 > pre2 {
		return 1
	}

	return 0
}

func cacheFilePath(cacheDir string) string {
	if cacheDir == "" {
		if home, err := install.HomeDir(""); err == nil {
			cacheDir = filepath.Join(home, "cache")
		} else {
			cacheDir = filepath.Join(os.TempDir(), "prumo-cache")
		}
	}
	return filepath.Join(cacheDir, "update_check.json")
}

func readCachedCheck(path string) (*CheckResult, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cached cachedCheck
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}
	if time.Now().After(cached.ExpiresAt) {
		return nil, false
	}
	res := cached.Result
	res.FromCache = true
	return &res, true
}

func writeCachedCheck(path string, res CheckResult, ttl time.Duration) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	cached := cachedCheck{
		Result:    res,
		ExpiresAt: time.Now().Add(ttl),
	}
	data, err := json.MarshalIndent(cached, "", "  ")
	if err == nil {
		_ = os.WriteFile(path, data, 0644)
	}
}

// Check queries GitHub Releases and checks if a newer version exists.
func Check(ctx context.Context, opts CheckOptions) (*CheckResult, error) {
	if opts.Repository == "" {
		opts.Repository = DefaultRepository
	}
	if opts.CurrentVersion == "" {
		opts.CurrentVersion = protocol.CLIVersion
	}
	cacheFile := cacheFilePath(opts.CacheDir)

	if !opts.SkipCache {
		if cached, ok := readCachedCheck(cacheFile); ok {
			cached.CurrentVersion = opts.CurrentVersion
			cached.HasUpdate = CompareSemver(cached.CurrentVersion, cached.LatestVersion) < 0
			return cached, nil
		}
	}

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "https://api.github.com/repos"
	}
	apiURL := fmt.Sprintf("%s/%s/releases/latest", strings.TrimRight(baseURL, "/"), opts.Repository)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Prumo-Harness/"+protocol.CLIVersion)

	token := opts.Token
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found for repository %s", opts.Repository)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %s", resp.Status)
	}

	var ghRelease GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&ghRelease); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub release payload: %w", err)
	}

	expectedAssetName := TargetAssetName(runtime.GOOS, runtime.GOARCH)
	var assetURL, checksumsURL string

	for _, asset := range ghRelease.Assets {
		if asset.Name == expectedAssetName {
			assetURL = asset.BrowserDownloadURL
		}
		if asset.Name == "checksums.txt" {
			checksumsURL = asset.BrowserDownloadURL
		}
	}

	normalizedLatest := NormalizeVersion(ghRelease.TagName)
	releaseInfo := &ReleaseInfo{
		Version:      normalizedLatest,
		TagName:      ghRelease.TagName,
		ReleaseNotes: ghRelease.Body,
		PublishedAt:  ghRelease.PublishedAt,
		AssetURL:     assetURL,
		AssetName:    expectedAssetName,
		ChecksumsURL: checksumsURL,
		Prerelease:   ghRelease.Prerelease,
		HTMLURL:      ghRelease.HTMLURL,
	}

	hasUpdate := CompareSemver(opts.CurrentVersion, normalizedLatest) < 0

	result := &CheckResult{
		CurrentVersion: opts.CurrentVersion,
		LatestVersion:  normalizedLatest,
		HasUpdate:      hasUpdate,
		Release:        releaseInfo,
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
		FromCache:      false,
	}

	writeCachedCheck(cacheFile, *result, DefaultCacheTTL)
	return result, nil
}

// DownloadAndVerify downloads the release binary and its checksums.txt,
// verifying the SHA-256 hash strictly before returning.
func DownloadAndVerify(ctx context.Context, client *http.Client, release *ReleaseInfo, tempDir string) (string, string, error) {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	if release.AssetURL == "" {
		return "", "", fmt.Errorf("no binary asset found for %s (%s)", release.AssetName, release.TagName)
	}
	if release.ChecksumsURL == "" {
		return "", "", fmt.Errorf("no checksums.txt found in release %s (supply chain invariant violated)", release.TagName)
	}

	// 1. Download checksums.txt
	reqCS, err := http.NewRequestWithContext(ctx, http.MethodGet, release.ChecksumsURL, nil)
	if err != nil {
		return "", "", err
	}
	reqCS.Header.Set("User-Agent", "Prumo-Harness/"+protocol.CLIVersion)

	respCS, err := client.Do(reqCS)
	if err != nil {
		return "", "", fmt.Errorf("failed to download checksums: %w", err)
	}
	defer respCS.Body.Close()
	if respCS.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to download checksums.txt, HTTP status: %s", respCS.Status)
	}

	checksumMap := map[string]string{}
	scanner := bufio.NewScanner(respCS.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			hash := parts[0]
			name := filepath.Base(parts[1])
			checksumMap[name] = hash
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("failed to parse checksums.txt: %w", err)
	}

	expectedSHA, ok := checksumMap[release.AssetName]
	if !ok {
		return "", "", fmt.Errorf("checksum for %s not found in release checksums.txt", release.AssetName)
	}

	// 2. Download binary asset
	reqBin, err := http.NewRequestWithContext(ctx, http.MethodGet, release.AssetURL, nil)
	if err != nil {
		return "", "", err
	}
	reqBin.Header.Set("User-Agent", "Prumo-Harness/"+protocol.CLIVersion)

	respBin, err := client.Do(reqBin)
	if err != nil {
		return "", "", fmt.Errorf("failed to download binary asset: %w", err)
	}
	defer respBin.Body.Close()
	if respBin.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to download binary asset, HTTP status: %s", respBin.Status)
	}

	destPath := filepath.Join(tempDir, release.AssetName)
	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", "", err
	}

	hasher := sha256.New()
	multiWriter := io.MultiWriter(out, hasher)

	if _, err := io.Copy(multiWriter, respBin.Body); err != nil {
		out.Close()
		_ = os.Remove(destPath)
		return "", "", fmt.Errorf("failed to write binary: %w", err)
	}
	out.Close()

	actualSHA := fmt.Sprintf("%x", hasher.Sum(nil))
	if !strings.EqualFold(actualSHA, expectedSHA) {
		_ = os.Remove(destPath)
		return "", "", fmt.Errorf("supply chain verification failure: expected SHA256 %s, got %s", expectedSHA, actualSHA)
	}

	return destPath, actualSHA, nil
}

// ResolveTargetBinaryPath determines the active binary location to be replaced.
func ResolveTargetBinaryPath(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}

	executable, err := os.Executable()
	if err == nil {
		executable, _ = filepath.EvalSymlinks(executable)
		// Avoid overwriting temporary go run test or build cache binaries
		if !strings.Contains(executable, "go-build") {
			return executable, nil
		}
	}

	// Fallback to installation manifest
	home, homeErr := install.HomeDir("")
	if homeErr == nil {
		manifest, err := install.LoadManifest(home)
		if err == nil && manifest.BinaryPath != "" {
			return manifest.BinaryPath, nil
		}
	}

	// Fallback to standard user local bin
	if userHome, err := os.UserHomeDir(); err == nil {
		return filepath.Join(userHome, ".local", "bin", "prumo"), nil
	}

	return "", fmt.Errorf("could not determine target binary location")
}

// Apply atomically replaces targetBinaryPath with the downloaded binary,
// creating a backup of the old binary.
func Apply(downloadedPath string, targetBinaryPath string) (string, error) {
	if err := os.Chmod(downloadedPath, 0755); err != nil {
		return "", fmt.Errorf("failed to set executable permissions: %w", err)
	}

	targetDir := filepath.Dir(targetBinaryPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}

	backupPath := targetBinaryPath + ".old"
	_ = os.Remove(backupPath)

	// If current binary exists, backup first
	if _, err := os.Stat(targetBinaryPath); err == nil {
		if err := os.Rename(targetBinaryPath, backupPath); err != nil {
			// On some OSes or cross-device mounts, rename may fail; try copy + delete
			if copyErr := copyFile(targetBinaryPath, backupPath); copyErr != nil {
				return "", fmt.Errorf("failed to backup current binary: %w", copyErr)
			}
			_ = os.Remove(targetBinaryPath)
		}
	}

	// Move downloaded binary into place
	if err := os.Rename(downloadedPath, targetBinaryPath); err != nil {
		// Fallback to copy if cross-filesystem
		if copyErr := copyFile(downloadedPath, targetBinaryPath); copyErr != nil {
			// Try rollback if backup exists
			if _, statErr := os.Stat(backupPath); statErr == nil {
				_ = os.Rename(backupPath, targetBinaryPath)
			}
			return "", fmt.Errorf("failed to install new binary: %w", copyErr)
		}
		_ = os.Remove(downloadedPath)
	}

	_ = os.Chmod(targetBinaryPath, 0755)
	return backupPath, nil
}

// Rollback restores a previous binary from backup.
func Rollback(targetBinaryPath, backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("no backup path provided")
	}
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file does not exist: %w", err)
	}
	_ = os.Remove(targetBinaryPath)
	return os.Rename(backupPath, targetBinaryPath)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// Execute performs the complete update procedure:
// Check -> Download -> Verify Checksum -> Apply -> Update Manifest.
func Execute(ctx context.Context, opts UpdateOptions) (*UpdateResult, error) {
	checkRes, err := Check(ctx, opts.CheckOptions)
	if err != nil {
		return nil, fmt.Errorf("update check failed: %w", err)
	}

	targetVersion := checkRes.LatestVersion
	if opts.TargetVersion != "" {
		targetVersion = NormalizeVersion(opts.TargetVersion)
	}

	if !opts.Force && !checkRes.HasUpdate && opts.TargetVersion == "" {
		return &UpdateResult{
			PreviousVersion: checkRes.CurrentVersion,
			UpdatedVersion:  checkRes.CurrentVersion,
			Success:         true,
			Message:         fmt.Sprintf("Prumo is already on the latest version (%s)", checkRes.CurrentVersion),
		}, nil
	}

	targetBin, err := ResolveTargetBinaryPath(opts.TargetBinaryPath)
	if err != nil {
		return nil, fmt.Errorf("target binary resolution failed: %w", err)
	}

	if opts.DryRun {
		return &UpdateResult{
			PreviousVersion: checkRes.CurrentVersion,
			UpdatedVersion:  targetVersion,
			BinaryPath:      targetBin,
			Success:         true,
			Message:         fmt.Sprintf("[Dry-run] Would upgrade from %s to %s at %s", checkRes.CurrentVersion, targetVersion, targetBin),
		}, nil
	}

	tempDir, err := os.MkdirTemp("", "prumo-update-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary update directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}

	downloadedBin, verifiedSHA, err := DownloadAndVerify(ctx, client, checkRes.Release, tempDir)
	if err != nil {
		return nil, fmt.Errorf("download and verification failed: %w", err)
	}

	backupPath, err := Apply(downloadedBin, targetBin)
	if err != nil {
		return nil, fmt.Errorf("failed to apply update: %w", err)
	}

	// Update installation manifest
	home, homeErr := install.HomeDir("")
	if homeErr == nil {
		manifest, err := install.LoadManifest(home)
		if err == nil {
			manifest.PrumoVersion = targetVersion
			manifest.BinaryPath = targetBin
			_ = install.SaveManifest(home, manifest)
		}
	}

	// Invalidate update check cache
	_ = os.Remove(cacheFilePath(opts.CacheDir))

	return &UpdateResult{
		PreviousVersion: checkRes.CurrentVersion,
		UpdatedVersion:  targetVersion,
		BinaryPath:      targetBin,
		BackupPath:      backupPath,
		VerifiedSHA256:  verifiedSHA,
		Success:         true,
		Message:         fmt.Sprintf("Successfully upgraded Prumo from %s to %s", checkRes.CurrentVersion, targetVersion),
	}, nil
}

// StartPeriodicChecker begins a background goroutine checking for GitHub updates
// at regular intervals (default 6h if interval <= 0).
func StartPeriodicChecker(ctx context.Context, interval time.Duration, opts CheckOptions, onUpdateAvailable func(*ReleaseInfo)) {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	go func() {
		// Initial check shortly after startup (e.g. 10 seconds)
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
			if res, err := Check(ctx, opts); err == nil && res.HasUpdate && res.Release != nil {
				if onUpdateAvailable != nil {
					onUpdateAvailable(res.Release)
				}
			}
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				opts.SkipCache = true // periodic background check fetches fresh release
				if res, err := Check(ctx, opts); err == nil && res.HasUpdate && res.Release != nil {
					if onUpdateAvailable != nil {
						onUpdateAvailable(res.Release)
					}
				}
			}
		}
	}()
}

// AutoUpgradeOnRelease checks and applies an upgrade safely when a new release is available.
func AutoUpgradeOnRelease(ctx context.Context, opts UpdateOptions) (*UpdateResult, error) {
	opts.SkipCache = true
	return Execute(ctx, opts)
}
