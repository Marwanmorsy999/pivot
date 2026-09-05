// Package github provides a minimal GitHub API client for pivot integrations.
package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const apiBase = "https://api.github.com"

// Client is a minimal GitHub REST API client.
type Client struct {
	token  string
	repo   string // "owner/repo"
	http   *http.Client
}

// Issue represents a GitHub issue.
type Issue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	State  string   `json:"state"`
	URL    string   `json:"html_url"`
	Labels []string `json:"-"` // populated from label objects in ListIssues
}

// labeledIssueRaw is used to decode GitHub label objects in list responses.
type labeledIssueRaw struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	URL    string `json:"html_url"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// HasLabel returns true if the issue already carries the given label name.
func (c *Client) HasLabel(issue Issue, label string) bool {
	for _, l := range issue.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// RemoveLabel removes a label from an issue (no-op if label is absent).
func (c *Client) RemoveLabel(issueNumber int, label string) error {
	resp, err := c.do("DELETE",
		fmt.Sprintf("/repos/%s/issues/%d/labels/%s", c.repo, issueNumber, label),
		"")
	if err != nil {
		return fmt.Errorf("remove label: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	// 404 is fine — label was already absent.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("GitHub API %d removing label", resp.StatusCode)
	}
	return nil
}

// New creates a GitHub client. Token is read from GITHUB_TOKEN env if empty.
func New(token, repo string) (*Client, error) {
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("GitHub token required: set GITHUB_TOKEN or pass --github-token")
	}
	if repo == "" {
		// Try to detect from git remote
		repo = detectRepo()
	}
	if repo == "" {
		return nil, fmt.Errorf("GitHub repo required: set --github-repo owner/repo or run from a git repo")
	}
	return &Client{
		token: token,
		repo:  repo,
		http:  &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (c *Client) do(method, path string, body string) (*http.Response, error) {
	url := apiBase + path // #nosec G107 -- path is constructed from validated constants
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body)) // #nosec G107
	} else {
		req, err = http.NewRequest(method, url, nil) // #nosec G107
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.http.Do(req)
}

// GetIssue fetches a single issue by number.
func (c *Client) GetIssue(number int) (*Issue, error) {
	resp, err := c.do("GET", fmt.Sprintf("/repos/%s/issues/%d", c.repo, number), "")
	if err != nil {
		return nil, fmt.Errorf("fetch issue: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API %d for issue #%d", resp.StatusCode, number)
	}
	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("decode issue: %w", err)
	}
	return &issue, nil
}

// CreateComment posts a comment on an issue.
func (c *Client) CreateComment(issueNumber int, body string) error {
	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return fmt.Errorf("marshal comment: %w", err)
	}
	resp, err := c.do("POST",
		fmt.Sprintf("/repos/%s/issues/%d/comments", c.repo, issueNumber),
		string(payload))
	if err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("GitHub API %d creating comment", resp.StatusCode)
	}
	return nil
}

// AddLabel adds a label to an issue (creates the label if missing).
func (c *Client) AddLabel(issueNumber int, label string) error {
	payload, err := json.Marshal(map[string][]string{"labels": {label}})
	if err != nil {
		return fmt.Errorf("marshal label: %w", err)
	}
	resp, err := c.do("POST",
		fmt.Sprintf("/repos/%s/issues/%d/labels", c.repo, issueNumber),
		string(payload))
	if err != nil {
		return fmt.Errorf("add label: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API %d adding label", resp.StatusCode)
	}
	return nil
}

// CloseIssue closes an issue.
func (c *Client) CloseIssue(issueNumber int) error {
	payload, err := json.Marshal(map[string]string{"state": "closed"})
	if err != nil {
		return fmt.Errorf("marshal close: %w", err)
	}
	resp, err := c.do("PATCH",
		fmt.Sprintf("/repos/%s/issues/%d", c.repo, issueNumber),
		string(payload))
	if err != nil {
		return fmt.Errorf("close issue: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API %d closing issue", resp.StatusCode)
	}
	return nil
}

// ListIssues returns open issues with the given label.
func (c *Client) ListIssues(label string) ([]Issue, error) {
	path := fmt.Sprintf("/repos/%s/issues?state=open&labels=%s&per_page=50", c.repo, url.QueryEscape(label))
	resp, err := c.do("GET", path, "")
	if err != nil {
		return nil, fmt.Errorf("list issues: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API %d listing issues", resp.StatusCode)
	}
	var raw []labeledIssueRaw
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode issues: %w", err)
	}
	issues := make([]Issue, len(raw))
	for i, r := range raw {
		issues[i] = Issue{Number: r.Number, Title: r.Title, Body: r.Body, State: r.State, URL: r.URL}
		for _, l := range r.Labels {
			issues[i].Labels = append(issues[i].Labels, l.Name)
		}
	}
	return issues, nil
}

// IssueGoal builds a goal string from an issue for use as a pivot goal.
func IssueGoal(issue *Issue) string {
	goal := fmt.Sprintf("Resolve GitHub issue #%d: %s", issue.Number, issue.Title)
	if issue.Body != "" {
		body := issue.Body
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		goal += "\n\n" + body
	}
	return goal
}

// detectRepo tries to parse "owner/repo" from git remote.origin.url.
// It only reads the [remote "origin"] section to avoid picking up upstream remotes.
func detectRepo() string {
	data, err := os.ReadFile(".git/config") // #nosec G304 -- reads local repo config
	if err != nil {
		return ""
	}
	inOrigin := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		// Section headers
		if strings.HasPrefix(trimmed, "[") {
			inOrigin = trimmed == `[remote "origin"]`
			continue
		}
		if !inOrigin {
			continue
		}
		if !strings.HasPrefix(trimmed, "url = ") {
			continue
		}
		u := strings.TrimPrefix(trimmed, "url = ")
		// SSH: git@github.com:owner/repo.git
		if strings.HasPrefix(u, "git@github.com:") {
			u = strings.TrimPrefix(u, "git@github.com:")
			return strings.TrimSuffix(u, ".git")
		}
		// HTTPS: https://github.com/owner/repo.git
		if strings.Contains(u, "github.com/") {
			parts := strings.SplitN(u, "github.com/", 2)
			if len(parts) == 2 {
				return strings.TrimSuffix(parts[1], ".git")
			}
		}
	}
	return ""
}
