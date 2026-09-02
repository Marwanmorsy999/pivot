package core

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Marwanmorsy999/pivot/internal/planner"
)

// newTestExecutor returns an executor with a nil State (logging is no-op in tests).
func newTestExecutor() *Executor {
	return &Executor{
		State:     nil,
		SessionID: "test-session",
		Provider:  "ollama",
		Model:     "llama3.2:3b",
		Outputs:   make(map[string]string),
		mu:        make(chan struct{}, 1),
	}
}

func init() {
	// Seed semaphore so tests can call lock/unlock.
	// (done in NewExecutor; we replicate it for newTestExecutor)
}

func withSem(e *Executor) *Executor {
	e.mu <- struct{}{}
	return e
}

// stubTask builds a minimal Task for testing.
func stubTask(id, tool string, args []string, deps []string) *Task {
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

func TestExecutor_ResolveArgs_LegacyOUTPUT(t *testing.T) {
	e := withSem(newTestExecutor())
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

func TestExecutor_ResolveArgs_NamedOUTPUT(t *testing.T) {
	e := withSem(newTestExecutor())
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

func TestExecutor_ResolveArgs_MissingNamedDep(t *testing.T) {
	e := withSem(newTestExecutor())
	task := stubTask("t", "echo", []string{"$OUTPUT[missing]"}, []string{"missing"})
	_, err := e.resolveArgs(task)
	if err == nil {
		t.Fatal("expected error for missing dep output")
	}
}

func TestExecutor_ResolveArgs_NoDepForOUTPUT(t *testing.T) {
	e := withSem(newTestExecutor())
	task := stubTask("t", "echo", []string{"$OUTPUT"}, nil)
	_, err := e.resolveArgs(task)
	if err == nil || !strings.Contains(err.Error(), "no dependencies") {
		t.Fatalf("expected no-dep error, got: %v", err)
	}
}

func TestExecutor_CommandForTool_Allowed(t *testing.T) {
	allowed := []string{"echo", "grep", "curl", "git", "jq", "sh", "bash", "python3", "docker", "aws"}
	for _, tool := range allowed {
		_, err := commandForTool(context.Background(), tool, nil)
		if err != nil {
			t.Errorf("tool %q should be allowed, got: %v", tool, err)
		}
	}
}

func TestExecutor_CommandForTool_Denied(t *testing.T) {
	denied := []string{"nc", "ncat", "socat", "python2", "perl", "ruby", "php"}
	for _, tool := range denied {
		_, err := commandForTool(context.Background(), tool, nil)
		if err == nil {
			t.Errorf("tool %q should be denied but was allowed", tool)
		}
	}
}

func TestExecutor_Timeout(t *testing.T) {
	e := withSem(newTestExecutor())
	eventCh := make(chan Event, 10)

	// "sleep 10" with a 100ms timeout — should fail fast.
	task := &Task{
		Task: planner.Task{
			ID:         "slow",
			Type:       planner.TypeTool,
			Tool:       "sleep",
			Args:       []string{"10"},
			TimeoutSec: 0, // will use default — override below
		},
		Status: "pending",
	}
	task.TimeoutSec = 0 // force defaultTimeoutSec path

	// Use a context that expires quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	// sleep is not in our allowlist — use sh instead
	task.Tool = "sh"
	task.Args = []string{"-c", "sleep 10"}

	err := e.Execute(ctx, task, eventCh)
	elapsed := time.Since(start)

	// Should have returned well before 10s
	if elapsed > 3*time.Second {
		t.Fatalf("Execute took %v — timeout not respected", elapsed)
	}
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func TestExecutor_SetGetOutput_Concurrent(t *testing.T) {
	e := withSem(newTestExecutor())
	done := make(chan struct{})

	// Write and read concurrently — race detector will catch issues.
	go func() {
		for i := 0; i < 100; i++ {
			e.SetOutput("key", "value")
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		e.GetOutput("key")
	}
	<-done
}

func TestExecutor_ParseTasks_StripsFences(t *testing.T) {
	raw := "```json\n{\"tasks\":[]}\n```"
	tasks, err := parseTasks(raw)
	if err != nil {
		t.Fatalf("parseTasks failed: %v", err)
	}
	if tasks == nil {
		t.Fatal("expected non-nil tasks slice")
	}
}

func TestExecutor_ParseTasks_PlainJSON(t *testing.T) {
	raw := `{"tasks":[{"id":"a","type":"tool","tool":"echo","args":[],"depends_on":[]}]}`
	tasks, err := parseTasks(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ID != "a" {
		t.Fatalf("unexpected tasks: %v", tasks)
	}
}
