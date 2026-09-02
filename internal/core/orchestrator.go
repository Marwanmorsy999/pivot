package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/Marwanmorsy999/pivot/internal/planner"
	"github.com/Marwanmorsy999/pivot/internal/state"
)

type EventType string

const (
	EventTaskUpdate EventType = "task_update"
	EventComplete   EventType = "complete"
	EventError      EventType = "error"
)

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

type Orchestrator struct {
	tasks     []planner.Task
	sessionID string
	state     *state.State
	eventCh   chan Event
}

func NewOrchestrator(tasks []planner.Task, sessionID string, s *state.State, eventCh chan Event) *Orchestrator {
	return &Orchestrator{
		tasks:     tasks,
		sessionID: sessionID,
		state:     s,
		eventCh:   eventCh,
	}
}

func (o *Orchestrator) Run(ctx context.Context) error {
	coreTasks := make([]*Task, len(o.tasks))
	for i, t := range o.tasks {
		coreTasks[i] = &Task{Task: t, Status: "pending"}
	}

	graph := NewGraph(coreTasks)
	order, err := graph.Order()
	if err != nil {
		o.eventCh <- Event{Type: EventError, Message: err.Error()}
		return err
	}

	executor := NewExecutor(o.sessionID, o.state)

	var mu sync.Mutex
	taskMap := make(map[string]*Task)
	for _, t := range coreTasks {
		taskMap[t.ID] = t
	}

	var firstErr error
	for _, id := range order {
		select {
		case <-ctx.Done():
			o.eventCh <- Event{Type: EventError, Message: "interrupted"}
			return ctx.Err()
		default:
		}

		task := graph.GetTask(id)

		completed, err := o.state.IsTaskCompleted(o.sessionID, id)
		if err != nil {
			return fmt.Errorf("check task %s completion: %w", id, err)
		}
		if completed {
			task.Status = "skipped"
			o.eventCh <- Event{Type: EventTaskUpdate, TaskID: id, Status: "skipped"}
			continue
		}

		mu.Lock()
		depFailed := false
		for _, depID := range task.DependsOn {
			depTask := taskMap[depID]
			if depTask.Status == "failed" {
				depFailed = true
				skipErr := fmt.Errorf("dependency %s failed, skipping %s", depID, id)
				task.Status = "skipped"
				o.eventCh <- Event{Type: EventTaskUpdate, TaskID: id, Status: "skipped", Error: skipErr.Error()}
				if firstErr == nil {
					firstErr = skipErr
				}
				break
			}
		}
		mu.Unlock()

		if depFailed {
			// continue to next task — don't abort independent branches
			continue
		}

		if err := executor.Execute(ctx.Done(), task, o.eventCh); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			// continue: let independent tasks still run
			continue
		}
	}

	totalCost := executor.Cost
	totalTokens := executor.Tokens
	o.eventCh <- Event{
		Type:    EventComplete,
		Cost:    totalCost,
		Tokens:  totalTokens,
		Message: fmt.Sprintf("✅ Completed! Total cost: $%.6f, Tokens: %d", totalCost, totalTokens),
	}

	return firstErr
}
