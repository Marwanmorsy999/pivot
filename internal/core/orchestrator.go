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

// OrchestratorOptions configures runtime behaviour.
type OrchestratorOptions struct {
	// MaxParallel is the maximum number of tasks that may run concurrently.
	// 0 means unlimited (run all tasks in a wave at once).
	MaxParallel int
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

// Run executes the task graph using parallel wave scheduling.
//
// Tasks are grouped into waves by topological depth — tasks with no mutual
// dependencies are placed in the same wave and executed concurrently.  A
// configurable semaphore limits the maximum number of concurrent tasks.
//
// If a task fails its downstream dependents are skipped, but other independent
// tasks in later waves still execute.  Run returns the first error encountered
// (or nil on full success).
func (o *Orchestrator) Run(ctx context.Context) error {
	// Build internal task map.
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

	// Build task map for status lookups.
	taskMap := make(map[string]*Task, len(coreTasks))
	for _, t := range coreTasks {
		taskMap[t.ID] = t
	}
	var taskMapMu sync.RWMutex

	executor := NewExecutor(o.sessionID, "", "", o.state) // provider/model populated below

	// Semaphore for MaxParallel.
	var sem chan struct{}
	if o.opts.MaxParallel > 0 {
		sem = make(chan struct{}, o.opts.MaxParallel)
	}

	var firstErr atomic.Value // stores error

	for _, wave := range waves {
		select {
		case <-ctx.Done():
			o.eventCh <- Event{Type: EventError, Message: "interrupted"}
			return ctx.Err()
		default:
		}

		// Filter wave: skip already-completed and dep-failed tasks.
		runnable := make([]string, 0, len(wave))
		for _, id := range wave {
			task := graph.GetTask(id)

			done, err := o.state.IsTaskCompleted(o.sessionID, id)
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
				if taskMap[dep].Status == "failed" || taskMap[dep].Status == "skipped" {
					depFailed = true
					break
				}
			}
			taskMapMu.RUnlock()

			if depFailed {
				task.Status = "skipped"
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

		// Execute all runnable tasks in this wave concurrently.
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
