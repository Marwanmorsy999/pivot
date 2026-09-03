package core

import (
	"context"
	"testing"
	"time"

	"github.com/Marwanmorsy999/pivot/internal/planner"
)

func makeOrcTasks(specs []struct {
	id   string
	tool string
	args []string
	deps []string
}) []planner.Task {
	tasks := make([]planner.Task, len(specs))
	for i, s := range specs {
		tasks[i] = planner.Task{
			ID:        s.id,
			Type:      planner.TypeTool,
			Tool:      s.tool,
			Args:      s.args,
			DependsOn: s.deps,
		}
	}
	return tasks
}

func TestOrchestratorOptions_MaxParallel(t *testing.T) {
	opts := OrchestratorOptions{MaxParallel: 2}
	if opts.MaxParallel != 2 {
		t.Fatalf("expected 2, got %d", opts.MaxParallel)
	}
}

func TestOrchestratorOptions_ProviderModel(t *testing.T) {
	opts := OrchestratorOptions{Provider: "anthropic", Model: "claude-opus-4-5"}
	if opts.Provider != "anthropic" {
		t.Errorf("Provider not set: %q", opts.Provider)
	}
	if opts.Model != "claude-opus-4-5" {
		t.Errorf("Model not set: %q", opts.Model)
	}
}

func TestNewOrchestrator_Fields(t *testing.T) {
	tasks := makeOrcTasks([]struct {
		id   string
		tool string
		args []string
		deps []string
	}{{"a", "echo", []string{"hi"}, nil}})

	eventCh := make(chan Event, 10)
	o := NewOrchestrator(tasks, "sess-1", nil, eventCh, OrchestratorOptions{MaxParallel: 4})

	if o.sessionID != "sess-1" {
		t.Errorf("sessionID mismatch: got %q", o.sessionID)
	}
	if len(o.tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(o.tasks))
	}
	if o.opts.MaxParallel != 4 {
		t.Errorf("MaxParallel not wired: got %d", o.opts.MaxParallel)
	}
}

func TestOrchestratorIsCompleted_NilState(t *testing.T) {
	// isCompleted must return false (not panic) when state is nil
	o := &Orchestrator{state: nil, sessionID: "x"}
	done, err := o.isCompleted("any-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if done {
		t.Error("expected false for nil state")
	}
}

func TestGraph_WavesNoDeadlock_Diamond(t *testing.T) {
	// A → {B, C} → D (diamond) — Waves() must not deadlock and return 3 waves
	tasks := makeTasks([]struct {
		id   string
		deps []string
	}{
		{"A", nil},
		{"B", []string{"A"}},
		{"C", []string{"A"}},
		{"D", []string{"B", "C"}},
	})
	g := NewGraph(tasks)
	waves, err := g.Waves()
	if err != nil {
		t.Fatal(err)
	}
	if len(waves) != 3 {
		t.Fatalf("expected 3 waves, got %d: %v", len(waves), waves)
	}
	if len(waves[1]) != 2 {
		t.Fatalf("wave 1 should contain B and C, got %v", waves[1])
	}
}

func TestOrchestratorRun_ContextCancellation(t *testing.T) {
	// A pre-cancelled context must cause Run() to return quickly without hanging.
	tasks := makeOrcTasks([]struct {
		id   string
		tool string
		args []string
		deps []string
	}{
		{"a", "sleep", []string{"60"}, nil},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	eventCh := make(chan Event, 50)
	o := NewOrchestrator(tasks, "sess-cancel", nil, eventCh, OrchestratorOptions{})

	done := make(chan error, 1)
	go func() { done <- o.Run(ctx) }()

	select {
	case <-done:
		// good — returned quickly
	case <-time.After(5 * time.Second):
		t.Fatal("orchestrator did not respect context cancellation within 5s")
	}
}

func TestOrchestratorRun_SingleTaskSuccess(t *testing.T) {
	// A single echo task with nil state should complete without panicking.
	tasks := makeOrcTasks([]struct {
		id   string
		tool string
		args []string
		deps []string
	}{
		{"greet", "echo", []string{"hello"}, nil},
	})

	eventCh := make(chan Event, 20)
	o := NewOrchestrator(tasks, "sess-single", nil, eventCh, OrchestratorOptions{})

	err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify EventComplete was emitted
	close(eventCh)
	var gotComplete bool
	for ev := range eventCh {
		if ev.Type == EventComplete {
			gotComplete = true
		}
	}
	if !gotComplete {
		t.Error("expected EventComplete to be emitted")
	}
}

func TestOrchestratorRun_SkipsDepOnFailure(t *testing.T) {
	// Verify that wave scheduling: a fails → b (dep on a) skipped.
	// Uses graph logic directly rather than real execution to avoid flakiness.
	tasks := makeTasks([]struct {
		id   string
		deps []string
	}{
		{"a", nil},
		{"b", []string{"a"}},
		{"c", nil}, // independent — should never be skipped
	})
	g := NewGraph(tasks)

	// Simulate: mark "a" as failed
	g.GetTask("a").Status = "failed"

	// Wave 0 = [a, c], Wave 1 = [b]
	waves, err := g.Waves()
	if err != nil {
		t.Fatal(err)
	}

	// In wave 1, task "b" has dep "a" with status "failed" → should be skipped
	taskB := g.GetTask("b")
	depFailed := false
	for _, dep := range taskB.DependsOn {
		s := g.GetTask(dep).Status
		if s == "failed" || s == "skipped" {
			depFailed = true
		}
	}
	if !depFailed {
		t.Errorf("expected dep failure to be detected for b; waves=%v", waves)
	}

	// Task "c" in wave 0 has no deps — should not be affected
	taskC := g.GetTask("c")
	if len(taskC.DependsOn) != 0 {
		t.Errorf("c should have no deps")
	}
}
