package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RepositoryHost interface {
	RepositoryState() (RepositoryState, error)
	Rulesets() ([]RulesetState, error)
	Labels(name string) ([]LabelState, error)
	CreateCheckRun(check CheckRunRequest) (*CheckRunResponse, error)
}

type RepositoryState struct {
	DefaultBranch    string `json:"default_branch"`
	SquashMerge      bool   `json:"squash_merge"`
	MergeCommits     bool   `json:"merge_commits"`
	RebaseMerge      bool   `json:"rebase_merge"`
	DeleteAfterMerge bool   `json:"delete_after_merge"`
}

type RulesetState struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Target      string `json:"target"`
	Enforcement string `json:"enforcement"`
}

type LabelState struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Client struct {
	RepositoryName string
	Token          string
	HTTP           *http.Client
	BaseURL        string
}

func NewAdapter(repository, token string) *Client {
	return &Client{RepositoryName: repository, Token: token, HTTP: &http.Client{Timeout: 20 * time.Second}, BaseURL: "https://api.github.com/repos"}
}

func (a *Client) request(method, path string, payload any) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(data)
	}
	baseURL := a.BaseURL
	if baseURL == "" {
		baseURL = "https://api.github.com/repos"
	}
	request, err := http.NewRequest(method, baseURL+"/"+a.RepositoryName+path, body)
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	if a.Token != "" {
		request.Header.Set("Authorization", "Bearer "+a.Token)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := a.HTTP.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	return data, response.StatusCode, nil
}

func (a *Client) RepositoryState() (RepositoryState, error) {
	var state RepositoryState
	data, status, err := a.request(http.MethodGet, "", nil)
	if err != nil {
		return state, err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return state, fmt.Errorf("permission denied reading repository")
	}
	if status == http.StatusNotFound {
		return state, fmt.Errorf("repository not found")
	}
	if status >= 300 {
		return state, fmt.Errorf("GitHub error %d", status)
	}
	var payload struct {
		DefaultBranch       string `json:"default_branch"`
		AllowSquashMerge    bool   `json:"allow_squash_merge"`
		AllowMergeCommit    bool   `json:"allow_merge_commit"`
		AllowRebaseMerge    bool   `json:"allow_rebase_merge"`
		DeleteBranchOnMerge bool   `json:"delete_branch_on_merge"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return state, err
	}
	state.DefaultBranch = payload.DefaultBranch
	state.SquashMerge = payload.AllowSquashMerge
	state.MergeCommits = payload.AllowMergeCommit
	state.RebaseMerge = payload.AllowRebaseMerge
	state.DeleteAfterMerge = payload.DeleteBranchOnMerge
	return state, nil
}

func (a *Client) UpdateRepositorySettings(settings map[string]any) error {
	_, status, err := a.request(http.MethodPatch, "", settings)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("GitHub error %d updating repository settings", status)
	}
	return nil
}

func (a *Client) CreateRuleset(payload map[string]any) error {
	_, status, err := a.request(http.MethodPost, "/rulesets", payload)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("GitHub error %d creating ruleset", status)
	}
	return nil
}

func (a *Client) Rulesets() ([]RulesetState, error) {
	data, status, err := a.request(http.MethodGet, "/rulesets", nil)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("GitHub error %d", status)
	}
	var rules []RulesetState
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (a *Client) Labels(name string) ([]LabelState, error) {
	var labels []LabelState
	page := 1
	for {
		data, status, err := a.request(http.MethodGet, fmt.Sprintf("/labels?per_page=100&page=%d", page), nil)
		if err != nil {
			return nil, err
		}
		if status >= 300 {
			return nil, fmt.Errorf("GitHub error %d", status)
		}
		var batch []LabelState
		if err := json.Unmarshal(data, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 || (name != "" && !strings.Contains(strings.ToLower(batch[0].Name), strings.ToLower(name))) {
			labels = append(labels, batch...)
			if len(batch) < 100 {
				break
			}
			page++
			continue
		}
		labels = append(labels, batch...)
		if len(batch) < 100 {
			break
		}
		page++
	}
	return labels, nil
}

type CheckRunOutput struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Text    string `json:"text,omitempty"`
}

type CheckRunRequest struct {
	Name       string          `json:"name"`
	HeadSHA    string          `json:"head_sha"`
	Status     string          `json:"status,omitempty"`     // queued, in_progress, completed
	Conclusion string          `json:"conclusion,omitempty"` // success, failure, neutral, cancelled, timed_out, action_required
	Output     *CheckRunOutput `json:"output,omitempty"`
}

type CheckRunResponse struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	HeadSHA    string `json:"head_sha"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HTMLURL    string `json:"html_url"`
}

func (a *Client) CreateCheckRun(check CheckRunRequest) (*CheckRunResponse, error) {
	data, status, err := a.request(http.MethodPost, "/check-runs", check)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("GitHub error %d creating check run: %s", status, strings.TrimSpace(string(data)))
	}
	var resp CheckRunResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
