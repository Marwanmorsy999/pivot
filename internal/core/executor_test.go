package core

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Marwanmorsy999/pivot/internal/planner"
)

// newTestExecutor returns an Executor with nil State (logging is a no-op).
func newTestExecutor() *Executor {
	e := &Executor{
		State:     nil,
		SessionID: "test-session",
		Provider:  "ollama",
		Model:     "llama3.2:3b",
		Outputs:   make(map[string]string),
		mu:        make(chan struct{}, 1),
	}
	e.mu <- struct{}{} // seed semaphore (mirrors NewExecutor)
	return e
}

func stubTask(id, tool string, args, deps []string) *Task {
	return &Task{
		Task: planner.Task{
			ID:        id,
			Type:      planner.TypeTool,
			Tool:      tool,
			Args:      args,
			DependsOn: deps,
		},
		Status: "pending",
	}
}

// ── resolveArgs ───────────────────────────────────────────────────────────────

func TestResolveArgs_LegacyOUTPUT(t *testing.T) {
	e := newTestExecutor()
	e.SetOutput("dep-1", "hello-world")
	task := stubTask("t", "echo", []string{"$OUTPUT"}, []string{"dep-1"})
	args, err := e.resolveArgs(task)
	if err != nil {
		t.Fatal(err)
	}
	if args[0] != "hello-world" {
		t.Fatalf("expected 'hello-world', got %q", args[0])
	}
}

func TestResolveArgs_NamedOUTPUT(t *testing.T) {
	e := newTestExecutor()
	e.SetOutput("step-a", "result-a")
	e.SetOutput("step-b", "result-b")
	task := stubTask("t", "echo",
		[]string{"$OUTPUT[step-a]", "$OUTPUT[step-b]"},
		[]string{"step-a", "step-b"},
	)
	args, err := e.resolveArgs(task)
	if err != nil {
		t.Fatal(err)
	}
	if args[0] != "result-a" {
		t.Fatalf("expected 'result-a', got %q", args[0])
	}
	if args[1] != "result-b" {
		t.Fatalf("expected 'result-b', got %q", args[1])
	}
}

func TestResolveArgs_MissingNamedDep(t *testing.T) {
	e := newTestExecutor()
	task := stubTask("t", "echo", []string{"$OUTPUT[missing]"}, []string{"missing"})
	_, err := e.resolveArgs(task)
	if err == nil {
		t.Fatal("expected error for missing dep output")
	}
}

func TestResolveArgs_NoDepForOUTPUT(t *testing.T) {
	e := newTestExecutor()
	task := stubTask("t", "echo", []string{"$OUTPUT"}, nil)
	_, err := e.resolveArgs(task)
	if err == nil || !strings.Contains(err.Error(), "no dependencies") {
		t.Fatalf("expected 'no dependencies' error, got: %v", err)
	}
}

func TestResolveArgs_NoSubstitution(t *testing.T) {
	e := newTestExecutor()
	task := stubTask("t", "echo", []string{"plain-arg", "--flag=value"}, nil)
	args, err := e.resolveArgs(task)
	if err != nil {
		t.Fatal(err)
	}
	if args[0] != "plain-arg" || args[1] != "--flag=value" {
		t.Fatalf("args should be unchanged: %v", args)
	}
}

// ── commandForTool ─────────────────────────────────────────────────────────────

func TestCommandForTool_Allowed(t *testing.T) {
	allowed := []string{"echo", "grep", "curl", "git", "jq", "sh", "bash", "python3", "docker", "aws", "sleep"}
	for _, tool := range allowed {
		_, err := commandForTool(context.Background(), tool, nil)
		if err != nil {
			t.Errorf("tool %q should be allowed, got: %v", tool, err)
		}
	}
}

func TestCommandForTool_Denied(t *testing.T) {
	denied := []string{"nc", "ncat", "socat", "perl", "ruby", "php", "notarealtool"}
	for _, tool := range denied {
		_, err := commandForTool(context.Background(), tool, nil)
		if err == nil {
			t.Errorf("tool %q should be denied", tool)
		}
	}
}

// ── SetOutput / GetOutput concurrency ──────────────────────────────────────────

func TestSetGetOutput_Concurrent(t *testing.T) {
	e := newTestExecutor()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 200; i++ {
			e.SetOutput("key", "value")
		}
		close(done)
	}()
	for i := 0; i < 200; i++ {
		e.GetOutput("key")
	}
	<-done
}

// ── Execute: pre-cancelled context ─────────────────────────────────────────────

func TestExecute_PreCancelledContext(t *testing.T) {
	e := newTestExecutor()
	eventCh := make(chan Event, 10)

	// Use a task with per-task timeout of 1s but cancel the parent ctx immediately.
	task := &Task{
		Task: planner.Task{
			ID:         "cancelled",
			Type:       planner.TypeTool,
			Tool:       "sh",
			Args:       []string{"-c", "sleep 5"},
			TimeoutSec: 1,
		},
		Status: "pending",
	}

	// Pre-cancel the context before calling Execute.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := e.Execute(ctx, task, eventCh)
	elapsed := time.Since(start)

	// Should return quickly (well under 1 second) because ctx is already done.
	if elapsed > 2*time.Second {
		t.Fatalf("Execute with pre-cancelled ctx took %v — should be near-instant", elapsed)
	}
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// ── Execute: nil state does not panic ──────────────────────────────────────────

func TestExecute_NilStateNoPanic(t *testing.T) {
	e := newTestExecutor()
	eventCh := make(chan Event, 10)
	task := stubTask("t", "echo", []string{"hello"}, nil)
	_ = e.Execute(context.Background(), task, eventCh)
}
