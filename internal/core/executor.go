package core

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Marwanmorsy999/pivot/internal/cost"
	"github.com/Marwanmorsy999/pivot/internal/state"
	"github.com/Marwanmorsy999/pivot/internal/worktree"
)

const (
	defaultTimeoutSec = 300 // 5 minutes per task
	defaultRetries    = 0
	retryBaseDelay    = time.Second
)

// Executor runs tasks and tracks outputs, cost, and tokens.
type Executor struct {
	State     *state.State
	SessionID string
	Provider  string
	Model     string

	mu      chan struct{} // protects Outputs, Cost, Tokens
	Outputs map[string]string
	Cost    float64
	Tokens  int
}

// NewExecutor creates an Executor for a session.
func NewExecutor(sessionID, provider, model string, s *state.State) *Executor {
	e := &Executor{
		State:     s,
		SessionID: sessionID,
		Provider:  provider,
		Model:     model,
		Outputs:   make(map[string]string),
		mu:        make(chan struct{}, 1),
	}
	e.mu <- struct{}{} // initialise semaphore as unlocked
	return e
}

func (e *Executor) lock()   { <-e.mu }
func (e *Executor) unlock() { e.mu <- struct{}{} }

// GetOutput safely returns the output of a completed task.
func (e *Executor) GetOutput(taskID string) (string, bool) {
	e.lock()
	out, ok := e.Outputs[taskID]
	e.unlock()
	return out, ok
}

// SetOutput safely records a task's output.
func (e *Executor) SetOutput(taskID, output string) {
	e.lock()
	e.Outputs[taskID] = output
	e.unlock()
}

// addCost safely accumulates cost/token totals.
func (e *Executor) addCost(tokens int, costUSD float64) {
	e.lock()
	e.Tokens += tokens
	e.Cost += costUSD
	e.unlock()
}

// outputRefRe matches $OUTPUT[task-id] references.
var outputRefRe = regexp.MustCompile(`\$OUTPUT\[([^\]]+)\]`)

// resolveArgs substitutes $OUTPUT[task-id] references in args.
// Also substitutes bare $OUTPUT with DependsOn[0] for backwards-compat.
func (e *Executor) resolveArgs(task *Task) ([]string, error) {
	args := make([]string, len(task.Args))
	for i, arg := range task.Args {
		resolved := arg

		// Named references: $OUTPUT[task-id]
		var resolveErr error
		resolved = outputRefRe.ReplaceAllStringFunc(resolved, func(match string) string {
			sub := outputRefRe.FindStringSubmatch(match)
			if len(sub) < 2 {
				return match
			}
			refID := sub[1]
			out, ok := e.GetOutput(refID)
			if !ok {
				resolveErr = fmt.Errorf("task %s: output of dependency %q not found", task.ID, refID)
				return match
			}
			return out
		})
		if resolveErr != nil {
			return nil, resolveErr
		}

		// Legacy bare $OUTPUT → DependsOn[0]
		if strings.Contains(resolved, "$OUTPUT") {
			if len(task.DependsOn) == 0 {
				return nil, fmt.Errorf("task %s uses $OUTPUT but has no dependencies", task.ID)
			}
			dep := task.DependsOn[0]
			out, ok := e.GetOutput(dep)
			if !ok {
				return nil, fmt.Errorf("task %s: dependency %q output not found", task.ID, dep)
			}
			resolved = strings.ReplaceAll(resolved, "$OUTPUT", out)
		}

		args[i] = resolved
	}
	return args, nil
}

// Execute runs a single task with timeout and retry semantics.
// It is safe to call concurrently from multiple goroutines.
func (e *Executor) Execute(ctx context.Context, task *Task, eventCh chan Event) error {
	args, err := e.resolveArgs(task)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		eventCh <- Event{Type: EventTaskUpdate, TaskID: task.ID, Status: "failed", Error: err.Error()}
		return err
	}

	timeoutSec := task.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = defaultTimeoutSec
	}
	maxRetries := task.Retries
	if maxRetries < 0 {
		maxRetries = 0
	}

	task.Status = "running"
	task.StartTime = time.Now().UnixMilli()
	eventCh <- Event{Type: EventTaskUpdate, TaskID: task.ID, Status: "running"}

	var lastErr error
	var outputStr string

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			eventCh <- Event{
				Type:    EventTaskUpdate,
				TaskID:  task.ID,
				Status:  "running",
				Message: fmt.Sprintf("retry %d/%d", attempt, maxRetries),
			}
		}

		taskCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		outputStr, lastErr = e.runOnce(taskCtx, task, args)
		cancel()

		if lastErr == nil {
			break
		}
	}

	tokenCount := cost.EstimateTokens(outputStr) + 100
	costUSD := cost.EstimateCostFlat(e.Provider, e.Model, tokenCount)
	e.addCost(tokenCount, costUSD)
	task.Cost = costUSD
	task.TokenUsed = tokenCount
	task.EndTime = time.Now().UnixMilli()

	if lastErr != nil {
		task.Status = "failed"
		task.Error = lastErr.Error()
		eventCh <- Event{Type: EventTaskUpdate, TaskID: task.ID, Status: "failed", Output: outputStr, Error: lastErr.Error()}
		_ = e.State.Log(state.JournalEntry{
			SessionID: e.SessionID, TaskID: task.ID, Tool: task.Tool,
			Args: args, Output: outputStr, Error: lastErr.Error(),
			Status: "failed", Cost: costUSD, Tokens: tokenCount,
		})
		return lastErr
	}

	task.Status = "completed"
	task.Output = outputStr
	e.SetOutput(task.ID, outputStr)
	eventCh <- Event{Type: EventTaskUpdate, TaskID: task.ID, Status: "completed", Output: outputStr}
	_ = e.State.Log(state.JournalEntry{
		SessionID: e.SessionID, TaskID: task.ID, Tool: task.Tool,
		Args: args, Output: outputStr,
		Status: "completed", Cost: costUSD, Tokens: tokenCount,
	})
	return nil
}

// runOnce executes the task command once, respecting the context deadline.
func (e *Executor) runOnce(ctx context.Context, task *Task, args []string) (string, error) {
	var (
		output []byte
		err    error
	)

	if task.Worktree && task.Type == "agent" {
		wt, wtErr := worktree.Create()
		if wtErr != nil {
			return "", fmt.Errorf("worktree creation failed: %w", wtErr)
		}
		defer func() {
			if cleanErr := worktree.Cleanup(wt); cleanErr != nil {
				fmt.Fprintf(os.Stderr, "worktree cleanup warning: %v\n", cleanErr)
			}
		}()
		cmd, cmdErr := commandForTool(ctx, task.Tool, append(args, "--cwd", wt))
		if cmdErr != nil {
			return "", cmdErr
		}
		cmd.Stderr = os.Stderr
		output, err = cmd.Output()
	} else {
		cmd, cmdErr := commandForTool(ctx, task.Tool, args)
		if cmdErr != nil {
			return "", cmdErr
		}
		cmd.Stderr = os.Stderr
		output, err = cmd.Output()
	}

	return strings.TrimSpace(string(output)), err
}

// commandForTool returns an exec.Cmd for the given tool name and args.
// The context is used for timeout/cancellation.
func commandForTool(ctx context.Context, tool string, args []string) (*exec.Cmd, error) {
	allowed := map[string]bool{
		// Unix core
		"find": true, "grep": true, "awk": true, "sed": true, "cat": true,
		"echo": true, "wc": true, "sort": true, "uniq": true, "head": true,
		"tail": true, "xargs": true, "tar": true, "zip": true, "unzip": true,
		"cut": true, "tr": true, "tee": true, "diff": true, "patch": true,
		"ls": true, "cp": true, "mv": true, "rm": true, "mkdir": true,
		"chmod": true, "chown": true, "touch": true, "stat": true, "file": true,
		"env": true, "printenv": true, "which": true, "date": true,
		// Shell passthrough
		"sh": true, "bash": true,
		// Network / data
		"jq": true, "curl": true, "wget": true, "ssh": true, "rsync": true,
		// Dev tools
		"git": true, "docker": true, "kubectl": true, "make": true,
		"python3": true, "python": true, "node": true, "go": true,
		"npm": true, "npx": true, "pip": true, "pip3": true,
		"cargo": true, "rustc": true,
		"terraform": true, "helm": true,
		// Cloud CLIs
		"aws": true, "gcloud": true, "az": true,
		// AI agents
		"ollama": true, "claude-code": true, "gemini-cli": true,
	}

	if !allowed[tool] {
		return nil, fmt.Errorf("unsupported executable: %q (add to allowlist if needed)", tool)
	}
	return exec.CommandContext(ctx, tool, args...), nil // #nosec G204 -- tool validated against allowlist above
}
