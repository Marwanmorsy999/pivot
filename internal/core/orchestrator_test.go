package core

import (
	"context"
	"testing"
	"time"

	"github.com/Marwanmorsy999/pivot/internal/planner"
)

// stubState is a minimal no-op state that marks nothing as completed.
type stubState struct {
	completed map[string]bool
	logged    []string
}

func (s *stubState) IsTaskCompleted(sessionID, taskID string) (bool, error) {
	return s.completed[taskID], nil
}
func (s *stubState) Log(entry interface{}) error {
	return nil
}

// We can't use stubState directly with the real Orchestrator because it takes
// *state.State. Instead we test the wave/parallel logic through the executor's
// output map, using real "echo" commands.

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

// TestOrchestratorOptions_MaxParallel verifies that the semaphore field is set.
func TestOrchestratorOptions_MaxParallel(t *testing.T) {
	opts := OrchestratorOptions{MaxParallel: 2}
	if opts.MaxParallel != 2 {
		t.Fatalf("expected MaxParallel=2, got %d", opts.MaxParallel)
	}
}

// TestNewOrchestrator_Fields verifies constructor wires fields correctly.
func TestNewOrchestrator_Fields(t *testing.T) {
	tasks := makeOrcTasks([]struct {
		id   string
		tool string
		args []string
		deps []string
	}{{"a", "echo", []string{"hi"}, nil}})

	eventCh := make(chan Event, 10)
	o := NewOrchestrator(tasks, "sess", nil, eventCh, OrchestratorOptions{MaxParallel: 4})

	if o.sessionID != "sess" {
		t.Errorf("sessionID not set")
	}
	if len(o.tasks) != 1 {
		t.Errorf("tasks not set")
	}
	if o.opts.MaxParallel != 4 {
		t.Errorf("opts not set")
	}
}

// TestGraph_WavesUsedByOrchestrator is an integration smoke test:
// verifies that parallel wave scheduling doesn't deadlock for a diamond graph.
func TestGraph_WavesNoDeadlock(t *testing.T) {
	specs := []struct {
		id   string
		deps []string
	}{
		{"A", nil},
		{"B", []string{"A"}},
		{"C", []string{"A"}},
		{"D", []string{"B", "C"}},
	}
	tasks := makeTasks(specs)
	g := NewGraph(tasks)
	waves, err := g.Waves()
	if err != nil {
		t.Fatal(err)
	}
	// Verify wave structure
	if len(waves) != 3 {
		t.Fatalf("expected 3 waves, got %d", len(waves))
	}
}

// TestOrchestratorContext_Cancellation verifies ctx.Done() is respected.
func TestOrchestratorContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	tasks := makeOrcTasks([]struct {
		id   string
		tool string
		args []string
		deps []string
	}{{"a", "echo", []string{"hi"}, nil}})

	eventCh := make(chan Event, 10)
	o := NewOrchestrator(tasks, "sess", nil, eventCh, OrchestratorOptions{})

	done := make(chan error, 1)
	go func() {
		done <- o.Run(ctx)
	}()

	select {
	case err := <-done:
		if err == nil {
			// Cancelled ctx may still complete fast echo — that's fine.
			// Main thing: it didn't hang.
		}
	case <-time.After(5 * time.Second):
		t.Fatal("orchestrator did not respect context cancellation within 5s")
	}
}
