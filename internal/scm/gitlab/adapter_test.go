package gitlab

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRepositoryStateAndProtectedBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v4/projects/group/project":
			w.Write([]byte(`{
				"default_branch": "main",
				"squash_option": "always",
				"merge_method": "ff",
				"remove_source_branch_after_merge": true
			}`))
		case "/api/v4/projects/group/project/protected_branches":
			w.Write([]byte(`[{"name":"main","push_access_level":40,"merge_access_level":40}]`))
		case "/api/v4/projects/group/project/labels":
			w.Write([]byte(`[{"name":"prumo-verified","color":"#00ff00"}]`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("PRIVATE-TOKEN") != "test-token" {
			t.Fatalf("missing or wrong PRIVATE-TOKEN header: %s", r.Header.Get("PRIVATE-TOKEN"))
		}
	}))
	defer server.Close()

	client := NewAdapter("group/project", "test-token")
	client.BaseURL = server.URL + "/api/v4"
	client.HTTP = server.Client()

	state, err := client.RepositoryState()
	if err != nil {
		t.Fatalf("RepositoryState failed: %v", err)
	}
	if state.DefaultBranch != "main" || state.SquashOption != "always" || !state.RemoveSourceBranchAfterMerge {
		t.Fatalf("unexpected state: %+v", state)
	}

	branches, err := client.ProtectedBranches()
	if err != nil || len(branches) != 1 || branches[0].Name != "main" {
		t.Fatalf("ProtectedBranches failed: %+v, err: %v", branches, err)
	}

	labels, err := client.Labels("prumo")
	if err != nil || len(labels) != 1 || labels[0].Name != "prumo-verified" {
		t.Fatalf("Labels failed: %+v, err: %v", labels, err)
	}
}

func TestPublishCommitStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/projects/group/project/statuses/deadbeef123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	client := NewAdapter("group/project", "test-token")
	client.BaseURL = server.URL + "/api/v4"
	client.HTTP = server.Client()

	err := client.PublishCommitStatus(CommitStatusRequest{
		SHA:         "deadbeef123",
		State:       "success",
		Name:        "prumo/verification",
		Description: "Passed deterministic gate",
	})
	if err != nil {
		t.Fatalf("PublishCommitStatus failed: %v", err)
	}
}

func TestRepositoryErrorsAreExplicit(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusInternalServerError} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		client := NewAdapter("group/project", "token")
		client.BaseURL = server.URL + "/api/v4"
		client.HTTP = server.Client()
		if _, err := client.RepositoryState(); err == nil {
			t.Fatalf("expected status %d error", status)
		}
		server.Close()
	}
}
