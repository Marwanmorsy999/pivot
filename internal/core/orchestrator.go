package core

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Marwanmorsy999/pivot/internal/planner"
	"github.com/Marwanmorsy999/pivot/internal/state"
)

// EventType classifies orchestration events sent to the TUI.
type EventType string

const (
	EventTaskUpdate EventType = "task_update"
	EventComplete   EventType = "complete"
	EventError      EventType = "error"
)

// Event carries task status updates and completion info to the TUI.
type Event struct {
	Type    EventType
	TaskID  string
	Status  string
	Output  string
	Error   string
	Message string
	Cost    float64
	Tokens  int
}

// OrchestratorOptions configures runtime behavior.
type OrchestratorOptions struct {
	// MaxParallel is the maximum number of tasks that may run concurrently.
	// 0 means unlimited (all tasks in a wave run at once).
	MaxParallel int
	// Provider and Model are forwarded to the Executor for cost tracking.
	Provider string
	Model    string
}

// Orchestrator drives execution of a task graph.
type Orchestrator struct {
	tasks     []planner.Task
	sessionID string
	state     *state.State
	eventCh   chan Event
	opts      OrchestratorOptions
}

// NewOrchestrator creates a new Orchestrator.
func NewOrchestrator(
	tasks []planner.Task,
	sessionID string,
	s *state.State,
	eventCh chan Event,
	opts OrchestratorOptions,
) *Orchestrator {
	return &Orchestrator{
		tasks:     tasks,
		sessionID: sessionID,
		state:     s,
		eventCh:   eventCh,
		opts:      opts,
	}
}

// isCompleted returns true if a task has already been recorded as completed in
// state. Returns false (not completed) when state is nil (e.g. in tests).
func (o *Orchestrator) isCompleted(taskID string) (bool, error) {
	if o.state == nil {
		return false, nil
	}
	return o.state.IsTaskCompleted(o.sessionID, taskID)
}

// Run executes the task graph using parallel wave scheduling.
//
// Tasks are grouped into waves by topological depth. Tasks within the same
// wave have no mutual dependencies and execute concurrently, bounded by the
// MaxParallel semaphore. Independent branches are never abandoned when a
// sibling or predecessor fails — only direct downstream dependents are skipped.
func (o *Orchestrator) Run(ctx context.Context) error {
	coreTasks := make([]*Task, len(o.tasks))
	for i, t := range o.tasks {
		coreTasks[i] = &Task{Task: t, Status: "pending"}
	}

	graph := NewGraph(coreTasks)
	waves, err := graph.Waves()
	if err != nil {
		o.eventCh <- Event{Type: EventError, Message: err.Error()}
		return err
	}

	taskMap := make(map[string]*Task, len(coreTasks))
	for _, t := range coreTasks {
		taskMap[t.ID] = t
	}
	var taskMapMu sync.RWMutex

	executor := NewExecutor(o.sessionID, o.opts.Provider, o.opts.Model, o.state)

	var sem chan struct{}
	if o.opts.MaxParallel > 0 {
		sem = make(chan struct{}, o.opts.MaxParallel)
	}

	var firstErr atomic.Value // stores the first error encountered

	for _, wave := range waves {
		select {
		case <-ctx.Done():
			o.eventCh <- Event{Type: EventError, Message: "interrupted"}
			return ctx.Err()
		default:
		}

		// Determine which tasks in this wave are actually runnable.
		runnable := make([]string, 0, len(wave))
		for _, id := range wave {
			task := graph.GetTask(id)

			done, err := o.isCompleted(id)
			if err != nil {
				return fmt.Errorf("check completion for %s: %w", id, err)
			}
			if done {
				task.Status = "skipped"
				o.eventCh <- Event{Type: EventTaskUpdate, TaskID: id, Status: "skipped"}
				continue
			}

			taskMapMu.RLock()
			depFailed := false
			for _, dep := range task.DependsOn {
				s := taskMap[dep].Status
				if s == "failed" || s == "skipped" {
					depFailed = true
					break
				}
			}
			taskMapMu.RUnlock()

			if depFailed {
				taskMapMu.Lock()
				task.Status = "skipped"
				taskMapMu.Unlock()
				o.eventCh <- Event{
					Type:    EventTaskUpdate,
					TaskID:  id,
					Status:  "skipped",
					Message: "dependency failed",
				}
				continue
			}

			runnable = append(runnable, id)
		}

		if len(runnable) == 0 {
			continue
		}

		// Run all tasks in this wave concurrently.
		var wg sync.WaitGroup
		for _, id := range runnable {
			wg.Add(1)
			go func(taskID string) {
				defer wg.Done()

				if sem != nil {
					sem <- struct{}{}
					defer func() { <-sem }()
				}

				task := graph.GetTask(taskID)
				if execErr := executor.Execute(ctx, task, o.eventCh); execErr != nil {
					firstErr.CompareAndSwap(nil, execErr)
					taskMapMu.Lock()
					taskMap[taskID].Status = "failed"
					taskMapMu.Unlock()
				}
			}(id)
		}
		wg.Wait()
	}

	o.eventCh <- Event{
		Type:    EventComplete,
		Cost:    executor.Cost,
		Tokens:  executor.Tokens,
		Message: fmt.Sprintf("✅ Done — cost $%.6f, tokens %d", executor.Cost, executor.Tokens),
	}

	if v := firstErr.Load(); v != nil {
		return v.(error)
	}
	return nil
}
