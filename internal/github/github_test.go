package github_test

import (
	"testing"

	"github.com/Marwanmorsy999/pivot/internal/github"
)

func TestNewClient_MissingToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	_, err := github.New("", "owner/repo")
	if err == nil {
		t.Error("expected error when no token provided")
	}
}

func TestNewClient_MissingRepo(t *testing.T) {
	_, err := github.New("fake-token", "")
	// Will either succeed with detected repo or fail — just should not panic
	_ = err
}

func TestNewClient_Valid(t *testing.T) {
	c, err := github.New("fake-token", "owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Error("expected non-nil client")
	}
}

func TestIssueGoal_TitleOnly(t *testing.T) {
	issue := &github.Issue{Number: 42, Title: "Fix the bug", Body: ""}
	goal := github.IssueGoal(issue)
	if goal == "" {
		t.Error("expected non-empty goal")
	}
	if !containsAll(goal, "#42", "Fix the bug") {
		t.Errorf("goal missing expected content: %q", goal)
	}
}

func TestIssueGoal_WithBody(t *testing.T) {
	issue := &github.Issue{Number: 7, Title: "Add feature", Body: "Some description here"}
	goal := github.IssueGoal(issue)
	if !containsAll(goal, "#7", "Add feature", "Some description here") {
		t.Errorf("goal missing body content: %q", goal)
	}
}

func TestIssueGoal_LongBodyTruncated(t *testing.T) {
	body := string(make([]byte, 1000))
	for i := range body {
		_ = i
	}
	longBody := ""
	for i := 0; i < 1000; i++ {
		longBody += "x"
	}
	issue := &github.Issue{Number: 1, Title: "T", Body: longBody}
	goal := github.IssueGoal(issue)
	if !containsAll(goal, "...") {
		t.Error("expected truncation marker for long body")
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
