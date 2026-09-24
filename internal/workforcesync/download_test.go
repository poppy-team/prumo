package workforcesync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// A skill package is a directory. Only manifest.json and SKILL.md were fetched,
// so every check, script, reference, template and example was silently dropped:
// the skill installed, the version matched, and the thing that does the work was
// missing (GAP-147).

// serveSkill stands up a server with one complete skill package.
func serveSkill(t *testing.T, skillID string, resources map[string][]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	base := "/src/prumo/resources/workforce/skills/" + skillID
	mux.HandleFunc(base+"/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"` + skillID + `","schema_version":3}`))
	})
	mux.HandleFunc(base+"/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("# " + skillID))
	})
	for dir, files := range resources {
		dir, files := dir, files
		mux.HandleFunc(base+"/"+dir+"/index.json", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"files": files})
		})
		for _, name := range files {
			name := name
			mux.HandleFunc(base+"/"+dir+"/"+name, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("content of " + dir + "/" + name))
			})
		}
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestASkillPackageArrivesWhole(t *testing.T) {
	server := serveSkill(t, "zoom-reflow", map[string][]string{
		"checks":     {"zoom-reflow-checklist.md"},
		"scripts":    {"verify.sh"},
		"references": {"zoom-reflow-guide.md"},
		"templates":  {"zoom-reflow-report.md"},
		"examples":   {"fluid-reflow.css"},
	})
	dest := filepath.Join(t.TempDir(), "zoom-reflow")
	s := &Service{HTTPClient: server.Client()}
	if err := s.downloadSkill(server.URL, "zoom-reflow", dest); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"manifest.json", "SKILL.md",
		"checks/zoom-reflow-checklist.md",
		"scripts/verify.sh",
		"references/zoom-reflow-guide.md",
		"templates/zoom-reflow-report.md",
		"examples/fluid-reflow.css",
	} {
		if _, err := os.Stat(filepath.Join(dest, rel)); err != nil {
			t.Errorf("the package arrived without %s: %v", rel, err)
		}
	}
}

func TestASkillWithoutResourceDirectoriesIsStillComplete(t *testing.T) {
	// A skill with no references/ is normal, and a 404 on the directory is how
	// the server says so. Refusing the package would make the common case a
	// failure.
	server := serveSkill(t, "tiny", nil)
	dest := filepath.Join(t.TempDir(), "tiny")
	s := &Service{HTTPClient: server.Client()}
	if err := s.downloadSkill(server.URL, "tiny", dest); err != nil {
		t.Fatalf("a skill with no resource directories failed to install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestARemoteCannotStealFilesThroughAListing(t *testing.T) {
	// A file name from a remote listing is joined onto the destination. A name
	// with a path separator in it is a remote choosing where the write lands.
	mux := http.NewServeMux()
	base := "/src/prumo/resources/workforce/skills/evil"
	mux.HandleFunc(base+"/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"evil"}`))
	})
	mux.HandleFunc(base+"/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("# evil"))
	})
	mux.HandleFunc(base+"/scripts/index.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"files": []string{"../../../escaped.sh"}})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	parent := t.TempDir()
	dest := filepath.Join(parent, "skill")
	s := &Service{HTTPClient: server.Client()}
	if err := s.downloadSkill(server.URL, "evil", dest); err == nil {
		t.Fatal("a listing with a traversal in it was accepted")
	}
	if _, err := os.Stat(filepath.Join(parent, "escaped.sh")); err == nil {
		t.Fatal("a remote wrote outside the skill directory")
	}
}
