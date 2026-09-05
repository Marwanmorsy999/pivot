package state

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestState(t *testing.T) *State {
	t.Helper()
	tmpDir := t.TempDir()
	pivotDir := filepath.Join(tmpDir, ".pivot")
	if err := os.MkdirAll(pivotDir, 0755); err != nil {
		t.Fatalf("failed to create pivot dir: %v", err)
	}
	t.Setenv("PIVOT_HOME", tmpDir)
	s, err := New()
	if err != nil {
		t.Fatalf("failed to create state: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGetSession(t *testing.T) {
	s := newTestState(t)
	id, err := s.CreateSession("test goal")
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty session id")
	}

	goal, err := s.GetGoal(id)
	if err != nil {
		t.Fatalf("GetGoal error: %v", err)
	}
	if goal != "test goal" {
		t.Errorf("expected 'test goal', got %q", goal)
	}
}

func TestGetSessions(t *testing.T) {
	s := newTestState(t)

	for i := 0; i < 3; i++ {
		_, err := s.CreateSession("goal")
		if err != nil {
			t.Fatalf("CreateSession error: %v", err)
		}
	}

	sessions, err := s.GetSessions()
	if err != nil {
		t.Fatalf("GetSessions error: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestLogAndIsTaskCompleted(t *testing.T) {
	s := newTestState(t)
	id, _ := s.CreateSession("test")

	err := s.Log(JournalEntry{
		SessionID: id,
		TaskID:    "a",
		Tool:      "grep",
		Args:      []string{"-r", "TODO", "."},
		Output:    "found TODOs",
		Status:    "completed",
		Cost:      0.001,
		Tokens:    500,
	})
	if err != nil {
		t.Fatalf("Log error: %v", err)
	}

	completed, err := s.IsTaskCompleted(id, "a")
	if err != nil {
		t.Fatalf("IsTaskCompleted error: %v", err)
	}
	if !completed {
		t.Error("expected task to be completed")
	}

	completed, err = s.IsTaskCompleted(id, "b")
	if err != nil {
		t.Fatalf("IsTaskCompleted error: %v", err)
	}
	if completed {
		t.Error("expected task b to not be completed")
	}
}

func TestGetFailedTasks(t *testing.T) {
	s := newTestState(t)
	id, _ := s.CreateSession("test")

	if err := s.Log(JournalEntry{SessionID: id, TaskID: "a", Status: "completed"}); err != nil {
		t.Fatalf("Log error: %v", err)
	}
	if err := s.Log(JournalEntry{SessionID: id, TaskID: "b", Status: "failed"}); err != nil {
		t.Fatalf("Log error: %v", err)
	}
	if err := s.Log(JournalEntry{SessionID: id, TaskID: "c", Status: "failed"}); err != nil {
		t.Fatalf("Log error: %v", err)
	}

	failed, err := s.GetFailedTasks(id)
	if err != nil {
		t.Fatalf("GetFailedTasks error: %v", err)
	}
	if len(failed) != 2 {
		t.Errorf("expected 2 failed tasks, got %d", len(failed))
	}
}

func TestGetFailedTasks_ExcludesRetrySuccess(t *testing.T) {
	s := newTestState(t)
	id, _ := s.CreateSession("retry test")

	// First attempt fails, second succeeds — should NOT appear in failed list.
	if err := s.Log(JournalEntry{SessionID: id, TaskID: "task1", Tool: "sh", Status: "failed"}); err != nil {
		t.Fatalf("Log: %v", err)
	}
	if err := s.Log(JournalEntry{SessionID: id, TaskID: "task1", Tool: "sh", Status: "completed"}); err != nil {
		t.Fatalf("Log: %v", err)
	}

	failed, err := s.GetFailedTasks(id)
	if err != nil {
		t.Fatalf("GetFailedTasks: %v", err)
	}
	for _, f := range failed {
		if f == "task1" {
			t.Error("task1 should not be in failed list — it ultimately succeeded")
		}
	}
}

func TestGetJournalEntries_LatestPerTask(t *testing.T) {
	s := newTestState(t)
	id, _ := s.CreateSession("dedup")

	// Two attempts for the same task — only last should be returned.
	_ = s.Log(JournalEntry{SessionID: id, TaskID: "t1", Tool: "sh", Status: "failed"})
	_ = s.Log(JournalEntry{SessionID: id, TaskID: "t1", Tool: "sh", Status: "completed", Output: "ok"})

	entries, err := s.GetJournalEntries(id)
	if err != nil {
		t.Fatalf("GetJournalEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 deduplicated entry, got %d", len(entries))
	}
	if entries[0].Status != "completed" {
		t.Errorf("expected completed entry, got %q", entries[0].Status)
	}
}

func TestUpdateSessionStatus(t *testing.T) {
	s := newTestState(t)
	id, _ := s.CreateSession("status test")

	if err := s.UpdateSessionStatus(id, "completed"); err != nil {
		t.Fatalf("UpdateSessionStatus: %v", err)
	}
	status, err := s.GetSessionStatus(id)
	if err != nil {
		t.Fatalf("GetSessionStatus: %v", err)
	}
	if status != "completed" {
		t.Errorf("expected completed, got %q", status)
	}
}

func TestDeleteSession_RemovesJournal(t *testing.T) {
	s := newTestState(t)
	id, _ := s.CreateSession("to delete")
	_ = s.Log(JournalEntry{SessionID: id, TaskID: "t1", Tool: "echo", Status: "completed"})

	if err := s.DeleteSession(id); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	entries, _ := s.GetJournalEntries(id)
	if len(entries) != 0 {
		t.Errorf("expected 0 journal entries after delete, got %d", len(entries))
	}
}

func TestGetSessionSummaries_Filter(t *testing.T) {
	s := newTestState(t)
	id1, _ := s.CreateSession("fail session")
	id2, _ := s.CreateSession("ok session")
	_ = s.UpdateSessionStatus(id1, "failed")
	_ = s.UpdateSessionStatus(id2, "completed")

	failed, err := s.GetSessionSummaries("failed")
	if err != nil {
		t.Fatalf("GetSessionSummaries: %v", err)
	}
	if len(failed) != 1 || failed[0].ID != id1 {
		t.Errorf("expected only id1 in failed list, got %+v", failed)
	}
}

func TestCreateSession_UniqueIDs(t *testing.T) {
	s := newTestState(t)
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		id, err := s.CreateSession("goal")
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		if seen[id] {
			t.Errorf("duplicate session ID generated: %s", id)
		}
		seen[id] = true
	}
}
