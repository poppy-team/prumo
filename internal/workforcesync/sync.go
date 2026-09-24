package workforcesync

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/install"
	"github.com/raillen/prumo/internal/packages"
	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/resolver"
)

var (
	ErrSkillNotFound   = errors.New("skill not found in catalog or remote")
	ErrOfflineRequired = errors.New("cannot download skill: offline mode enabled and package not cached")
)

type Options struct {
	ProjectRoot   string
	HomeDir       string
	RepoRoot      string
	RemoteBaseURL string
	Version       string
	OfflineOnly   bool
	ForceRemote   bool
	TargetSkills  []string
}

type Result struct {
	ResolvedSkills  []string            `json:"resolved_skills"`
	InstalledSkills []string            `json:"installed_skills"`
	CacheHits       []string            `json:"cache_hits"`
	RemoteDownloads []string            `json:"remote_downloads"`
	LockPackages    []packages.Resolved `json:"lock_packages"`
}

type Service struct {
	HTTPClient *http.Client
}

func NewService() *Service {
	return &Service{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) Sync(opts Options) (*Result, error) {
	if opts.ProjectRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		opts.ProjectRoot = cwd
	}
	if opts.HomeDir == "" {
		home, err := install.HomeDir("")
		if err != nil {
			return nil, err
		}
		opts.HomeDir = home
	}
	if opts.Version == "" {
		opts.Version = protocol.CLIVersion
	}
	if opts.RemoteBaseURL == "" {
		opts.RemoteBaseURL = fmt.Sprintf("https://raw.githubusercontent.com/raillen/prumo/v%s", opts.Version)
	}

	// 1. Determine skills to install
	skillsToInstall := opts.TargetSkills
	if len(skillsToInstall) == 0 {
		var profile resolver.Profile
		profilePath := filepath.Join(opts.ProjectRoot, "prumo.json")
		if p, err := resolver.LoadProfile(profilePath); err == nil {
			profile = p
		} else {
			profile = resolver.Profile{Raw: map[string]any{}}
		}

		catalogRoot := opts.RepoRoot
		if catalogRoot == "" {
			// Check if we are within the prumo repository
			if _, err := os.Stat("src/prumo/resources/catalog/catalog.json"); err == nil {
				catalogRoot = "."
			}
		}

		if catalogRoot != "" {
			catalog, err := resolver.LoadCatalog(catalogRoot)
			if err == nil {
				resolution := catalog.Resolve(profile)
				skillsToInstall = resolution.Skills
			}
		}

		// Always ensure core skills are included
		coreSkills := []string{"clean-code", "cognitive-clarity", "testing-quality"}
		present := map[string]bool{}
		for _, s := range skillsToInstall {
			present[s] = true
		}
		for _, c := range coreSkills {
			if !present[c] {
				skillsToInstall = append(skillsToInstall, c)
			}
		}
	}
	sort.Strings(skillsToInstall)

	result := &Result{
		ResolvedSkills:  skillsToInstall,
		InstalledSkills: []string{},
		CacheHits:       []string{},
		RemoteDownloads: []string{},
		LockPackages:    []packages.Resolved{},
	}

	targetDir := filepath.Join(opts.ProjectRoot, ".ai", "skills")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, err
	}

	for _, skillID := range skillsToInstall {
		cacheSkillDir := filepath.Join(opts.HomeDir, "cache", "workforce", opts.Version, "skills", skillID)
		manifestPath := filepath.Join(cacheSkillDir, "manifest.json")

		inCache := false
		if !opts.ForceRemote {
			if info, err := os.Stat(manifestPath); err == nil && !info.IsDir() {
				inCache = true
			}
		}

		if inCache {
			result.CacheHits = append(result.CacheHits, skillID)
		} else {
			// Populate cache from RepoRoot if available
			populated := false
			if opts.RepoRoot != "" {
				repoSkillDir := filepath.Join(opts.RepoRoot, "src", "prumo", "resources", "workforce", "skills", skillID)
				if info, err := os.Stat(repoSkillDir); err == nil && info.IsDir() {
					_ = copyDir(repoSkillDir, cacheSkillDir)
					populated = true
				}
			}

			// If not populated from repo, attempt remote download
			if !populated {
				if opts.OfflineOnly {
					return nil, fmt.Errorf("%w: %s", ErrOfflineRequired, skillID)
				}
				if err := s.downloadSkill(opts.RemoteBaseURL, skillID, cacheSkillDir); err != nil {
					// Fallback to local fallback if exists in relative directory
					fallbackDir := filepath.Join("src", "prumo", "resources", "workforce", "skills", skillID)
					if info, statErr := os.Stat(fallbackDir); statErr == nil && info.IsDir() {
						_ = copyDir(fallbackDir, cacheSkillDir)
					} else {
						return nil, fmt.Errorf("failed to fetch skill %q: %w", skillID, err)
					}
				} else {
					result.RemoteDownloads = append(result.RemoteDownloads, skillID)
				}
			}
		}

		// Copy from cache to project .ai/skills/<skillID>
		destSkillDir := filepath.Join(targetDir, skillID)
		if err := copyDir(cacheSkillDir, destSkillDir); err != nil {
			return nil, fmt.Errorf("failed to install skill %q into project: %w", skillID, err)
		}
		result.InstalledSkills = append(result.InstalledSkills, skillID)

		// The checksum covers the whole installed tree, not the manifest.
		//
		// It covered only manifest.json, so a changed SKILL.md, a replaced
		// script or an edited reference left the lock file reporting exactly the
		// same hash. The lock said "this is the version you asked for" about a
		// package whose contents had changed underneath it (GAP-147).
		sum, err := packages.ChecksumTree(destSkillDir)
		if err != nil {
			return nil, fmt.Errorf("checksumming installed skill %q: %w", skillID, err)
		}

		result.LockPackages = append(result.LockPackages, packages.Resolved{
			ID:       skillID,
			Version:  opts.Version,
			Checksum: sum,
			Type:     "skill",
		})
	}

	// Update .ai/skills/manifest.json
	skillsManifest := map[string]any{
		"version": opts.Version,
		"skills":  result.InstalledSkills,
		"updated": time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(skillsManifest, "", "  ")
	if err == nil {
		_ = os.WriteFile(filepath.Join(targetDir, "manifest.json"), append(data, '\n'), 0644)
	}

	// Update prumo.lock
	lock := packages.NewLock(result.LockPackages)
	lockData, err := json.MarshalIndent(lock, "", "  ")
	if err == nil {
		_ = os.WriteFile(filepath.Join(opts.ProjectRoot, "prumo.lock"), append(lockData, '\n'), 0644)
	}

	return result, nil
}

func (s *Service) downloadSkill(baseURL, skillID, destDir string) error {
	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	// A skill package is a directory, not two files. Only manifest.json and
	// SKILL.md were fetched, so every check, script, reference, template and
	// example was silently dropped: the skill installed, the version matched,
	// and the thing that does the work was missing (GAP-147).
	//
	// The manifest is read first and its declared resources decide what else
	// to fetch, so the file list comes from the package rather than from a list
	// hard-coded here that would drift the first time a skill grew a folder.
	root := strings.TrimRight(baseURL, "/") + "/src/prumo/resources/workforce/skills/" + skillID

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	if err := fetchFile(client, root+"/manifest.json", filepath.Join(destDir, "manifest.json")); err != nil {
		return err
	}
	if err := fetchFile(client, root+"/SKILL.md", filepath.Join(destDir, "SKILL.md")); err != nil {
		return err
	}

	// Read it back so a truncated or empty download is caught here rather than
	// at the first command that tries to parse it.
	if _, err := os.ReadFile(filepath.Join(destDir, "manifest.json")); err != nil {
		return fmt.Errorf("skill %q manifest unreadable after download: %w", skillID, err)
	}
	for _, dir := range skillResourceDirs {
		files, err := listRemoteDir(client, root+"/"+dir)
		if err != nil {
			// A skill without a references/ directory is normal; a server that
			// cannot be asked is not. The difference is whether the directory
			// exists, and a 404 is the only honest evidence of that.
			if isNotFound(err) {
				continue
			}
			return fmt.Errorf("listing %s of skill %q: %w", dir, skillID, err)
		}
		for _, name := range files {
			// A name from a remote listing is not a path this process should
			// join blindly.
			if name == "" || strings.ContainsAny(name, `/\\`) || name == "." || name == ".." {
				return fmt.Errorf("remote listed an unusable file name %q in %s of skill %q", name, dir, skillID)
			}
			dest := filepath.Join(destDir, dir, name)
			if err := fetchFile(client, root+"/"+dir+"/"+name, dest); err != nil {
				return err
			}
		}
	}
	return nil
}

// skillResourceDirs are the directories a Skill v3 package carries beside its
// manifest and SKILL.md. They are listed here because a package's shape is
// fixed by the format, not by whichever skill happens to exist; a directory that
// is absent is not an error.
var skillResourceDirs = []string{"checks", "scripts", "references", "templates", "examples"}

// fetchFile downloads one file, closing the body before returning.
//
// The old loop deferred the close, so a package with N files held N open
// response bodies until the function returned — and the same pattern in a
// caller that downloaded a whole catalogue would exhaust the process's file
// descriptors.
func fetchFile(client *http.Client, url, dest string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("remote responded with HTTP %d for %s", resp.StatusCode, url)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// listRemoteDir asks a remote server for the files in a skill directory.
//
// It reads a directory listing rather than guessing names, because a skill's
// contents are the package author's business. A server that serves files but no
// listing is reported as "cannot ask" rather than as "empty", so a sync never
// silently installs a package with its resources left behind.
func listRemoteDir(client *http.Client, url string) ([]string, error) {
	resp, err := client.Get(url + "/index.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errNotFound{url}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote responded with HTTP %d for %s", resp.StatusCode, url+"/index.json")
	}
	var doc struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	return doc.Files, nil
}

// errNotFound marks a resource the server does not have.
type errNotFound struct{ url string }

func (e errNotFound) Error() string { return "not found: " + e.url }

func isNotFound(err error) bool {
	var notFound errNotFound
	return errors.As(err, &notFound)
}

func copyDir(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		if info.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0644)
	})
}
