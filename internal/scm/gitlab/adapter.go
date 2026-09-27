package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RepositoryState represents normalized GitLab project repository configuration.
type RepositoryState struct {
	DefaultBranch                string `json:"default_branch"`
	SquashOption                 string `json:"squash_option"`
	MergeMethod                  string `json:"merge_method"`
	RemoveSourceBranchAfterMerge bool   `json:"remove_source_branch_after_merge"`
}

// LabelState represents a GitLab project label.
type LabelState struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ProtectedBranchState represents protected branch rule in GitLab.
type ProtectedBranchState struct {
	Name             string `json:"name"`
	PushAccessLevel  int    `json:"push_access_level"`
	MergeAccessLevel int    `json:"merge_access_level"`
}

// CommitStatusRequest represents a pipeline/commit status update in GitLab.
type CommitStatusRequest struct {
	SHA         string `json:"sha"`
	State       string `json:"state"` // pending, running, success, failed, canceled
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	TargetURL   string `json:"target_url,omitempty"`
}

// Client talks to GitLab v4 REST API.
type Client struct {
	ProjectPath string
	Token       string
	HTTP        *http.Client
	BaseURL     string
}

// NewAdapter creates a new GitLab SCM adapter.
func NewAdapter(projectPath, token string) *Client {
	return &Client{
		ProjectPath: projectPath,
		Token:       token,
		HTTP:        &http.Client{Timeout: 20 * time.Second},
		BaseURL:     "https://gitlab.com/api/v4",
	}
}

func (c *Client) projectPathEncoded() string {
	return url.PathEscape(c.ProjectPath)
}

func (c *Client) request(method, path string, payload any) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(data)
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://gitlab.com/api/v4"
	}
	fullURL := fmt.Sprintf("%s/projects/%s%s", baseURL, c.projectPathEncoded(), path)
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.Token)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

// RepositoryState queries project settings and branch defaults.
func (c *Client) RepositoryState() (RepositoryState, error) {
	var state RepositoryState
	data, status, err := c.request(http.MethodGet, "", nil)
	if err != nil {
		return state, err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return state, fmt.Errorf("permission denied reading gitlab project")
	}
	if status == http.StatusNotFound {
		return state, fmt.Errorf("gitlab project not found")
	}
	if status >= 300 {
		return state, fmt.Errorf("GitLab error %d", status)
	}
	var payload struct {
		DefaultBranch                string `json:"default_branch"`
		SquashOption                 string `json:"squash_option"`
		MergeMethod                  string `json:"merge_method"`
		RemoveSourceBranchAfterMerge bool   `json:"remove_source_branch_after_merge"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return state, err
	}
	state.DefaultBranch = payload.DefaultBranch
	state.SquashOption = payload.SquashOption
	state.MergeMethod = payload.MergeMethod
	state.RemoveSourceBranchAfterMerge = payload.RemoveSourceBranchAfterMerge
	return state, nil
}

// Labels fetches project labels matching optional name prefix.
func (c *Client) Labels(name string) ([]LabelState, error) {
	path := "/labels?per_page=100"
	if name != "" {
		path += "&search=" + url.QueryEscape(name)
	}
	data, status, err := c.request(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("GitLab error %d reading labels", status)
	}
	var labels []LabelState
	if err := json.Unmarshal(data, &labels); err != nil {
		return nil, err
	}
	return labels, nil
}

// ProtectedBranches fetches protected branch definitions.
func (c *Client) ProtectedBranches() ([]ProtectedBranchState, error) {
	data, status, err := c.request(http.MethodGet, "/protected_branches", nil)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("GitLab error %d reading protected branches", status)
	}
	var branches []ProtectedBranchState
	if err := json.Unmarshal(data, &branches); err != nil {
		return nil, err
	}
	return branches, nil
}

// PublishCommitStatus posts commit status (CI / Verification evidence) to a commit.
func (c *Client) PublishCommitStatus(req CommitStatusRequest) error {
	path := fmt.Sprintf("/statuses/%s", url.PathEscape(req.SHA))
	payload := map[string]any{
		"state": req.State,
		"name":  req.Name,
	}
	if req.Description != "" {
		payload["description"] = req.Description
	}
	if req.TargetURL != "" {
		payload["target_url"] = req.TargetURL
	}
	data, status, err := c.request(http.MethodPost, path, payload)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("GitLab error %d posting commit status: %s", status, strings.TrimSpace(string(data)))
	}
	return nil
}
