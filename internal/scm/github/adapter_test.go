package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRepositoryStateAndRulesets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/repos/example/project":
			writer.Write([]byte(`{"default_branch":"main","allow_squash_merge":true,"allow_merge_commit":false,"allow_rebase_merge":false,"delete_branch_on_merge":true}`))
		case "/repos/example/project/rulesets":
			writer.Write([]byte(`[{"id":1,"name":"main protection","target":"branch","enforcement":"active"}]`))
		case "/repos/example/project/labels":
			writer.Write([]byte(`[]`))
		default:
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
		if request.URL.Path != "/repos/example/project" && request.Header.Get("Authorization") == "" {
			t.Fatalf("missing authorization")
		}
	}))
	defer server.Close()
	client := NewAdapter("example/project", "token")
	client.BaseURL = server.URL + "/repos"
	client.HTTP = server.Client()
	state, err := client.RepositoryState()
	if err != nil {
		t.Fatalf("repository: %v", err)
	}
	if state.DefaultBranch != "main" || !state.SquashMerge || state.MergeCommits {
		t.Fatalf("payload mapping: %#v", state)
	}
	rules, err := client.Rulesets()
	if err != nil || len(rules) != 1 || rules[0].Name != "main protection" {
		t.Fatalf("rulesets: %#v %v", rules, err)
	}
}

func TestRepositoryErrorsAreExplicit(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusInternalServerError} {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(status)
		}))
		client := NewAdapter("example/project", "token")
		client.BaseURL = server.URL + "/repos"
		client.HTTP = server.Client()
		if _, err := client.RepositoryState(); err == nil {
			t.Fatalf("expected status %d error", status)
		}
		server.Close()
	}
}

func TestCreateCheckRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/example/project/check-runs" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"id": 123456,
			"name": "prumo/verification",
			"head_sha": "abc1234",
			"status": "completed",
			"conclusion": "success",
			"html_url": "https://github.com/example/project/runs/123456"
		}`))
	}))
	defer server.Close()

	client := NewAdapter("example/project", "token")
	client.BaseURL = server.URL + "/repos"
	client.HTTP = server.Client()

	resp, err := client.CreateCheckRun(CheckRunRequest{
		Name:       "prumo/verification",
		HeadSHA:    "abc1234",
		Status:     "completed",
		Conclusion: "success",
		Output: &CheckRunOutput{
			Title:   "Prumo Verification",
			Summary: "All verification gates passed",
		},
	})
	if err != nil {
		t.Fatalf("CreateCheckRun failed: %v", err)
	}
	if resp.ID != 123456 || resp.Status != "completed" || resp.Conclusion != "success" {
		t.Fatalf("unexpected check run response: %+v", resp)
	}
}
