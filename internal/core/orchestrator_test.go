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

// ── OrchestratorOptions ───────────────────────────────────────────────────────

func TestOrchestratorOptions_MaxParallel(t *testing.T) {
	opts := OrchestratorOptions{MaxParallel: 2}
	if opts.MaxParallel != 2 {
		t.Fatalf("expected 2, got %d", opts.MaxParallel)
	}
}

// ── NewOrchestrator ───────────────────────────────────────────────────────────

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

// ── Wave scheduling (via Graph) ───────────────────────────────────────────────

func TestGraph_WavesNoDeadlock_Diamond(t *testing.T) {
	// A → {B, C} → D  (classic diamond)
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

// ── Context cancellation ──────────────────────────────────────────────────────

func TestOrchestratorRun_ContextCancellation(t *testing.T) {
	// Orchestrator with nil state and a pre-cancelled context should
	// return quickly (not hang) and not panic.
	tasks := makeOrcTasks([]struct {
		id   string
		tool string
		args []string
		deps []string
	}{
		{"a", "sleep", []string{"60"}, nil},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Run is called

	eventCh := make(chan Event, 20)
	o := NewOrchestrator(tasks, "sess-cancel", nil, eventCh, OrchestratorOptions{})

	done := make(chan error, 1)
	go func() { done <- o.Run(ctx) }()

	select {
	case <-done:
		// returned — good
	case <-time.After(5 * time.Second):
		t.Fatal("orchestrator did not respect context cancellation within 5s")
	}
}

// ── Dep-failure isolation ─────────────────────────────────────────────────────

func TestOrchestratorRun_DepFailureSkipsDownstream(t *testing.T) {
	// a (bad command) → b depends on a → c is independent
	// Expected: b skipped, c runs, overall returns error
	tasks := makeOrcTasks([]struct {
		id   string
		tool string
		args []string
		deps []string
	}{
		{"a", "sh", []string{"-c", "exit 1"}, nil},
		{"b", "echo", []string{"$OUTPUT[a]"}, []string{"a"}},
		{"c", "echo", []string{"independent"}, nil},
	})

	eventCh := make(chan Event, 50)
	o := NewOrchestrator(tasks, "sess-depfail", nil, eventCh, OrchestratorOptions{MaxParallel: 4})

	err := o.Run(context.Background())
	// Overall run should return an error (a failed)
	if err == nil {
		t.Fatal("expected error when a task fails")
	}

	// Drain events and check c completed
	close(eventCh)
	statuses := map[string]string{}
	for ev := range eventCh {
		if ev.Type == EventTaskUpdate {
			statuses[ev.TaskID] = ev.Status
		}
	}
	if statuses["c"] != "completed" {
		t.Errorf("independent task c should have completed, got %q", statuses["c"])
	}
	if statuses["b"] != "skipped" {
		t.Errorf("downstream task b should be skipped, got %q", statuses["b"])
	}
}
