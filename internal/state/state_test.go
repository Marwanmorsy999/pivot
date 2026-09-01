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
	os.MkdirAll(pivotDir, 0755)
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

	s.Log(JournalEntry{SessionID: id, TaskID: "a", Status: "completed"})
	s.Log(JournalEntry{SessionID: id, TaskID: "b", Status: "failed"})
	s.Log(JournalEntry{SessionID: id, TaskID: "c", Status: "failed"})

	failed, err := s.GetFailedTasks(id)
	if err != nil {
		t.Fatalf("GetFailedTasks error: %v", err)
	}
	if len(failed) != 2 {
		t.Errorf("expected 2 failed tasks, got %d", len(failed))
	}
}
