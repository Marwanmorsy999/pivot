package core

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Marwanmorsy999/pivot/internal/cost"
	"github.com/Marwanmorsy999/pivot/internal/state"
	"github.com/Marwanmorsy999/pivot/internal/worktree"
)

type Executor struct {
	State     *state.State
	SessionID string
	Outputs   map[string]string
	Cost      float64
	Tokens    int
}

func NewExecutor(sessionID string, s *state.State) *Executor {
	return &Executor{
		State:     s,
		SessionID: sessionID,
		Outputs:   make(map[string]string),
	}
}

func (e *Executor) Execute(ctx <-chan struct{}, task *Task, eventCh chan Event) error {
	args := make([]string, len(task.Args))
	for i, arg := range task.Args {
		if strings.Contains(arg, "$OUTPUT") {
			if len(task.DependsOn) == 0 {
				return fmt.Errorf("task %s uses $OUTPUT but has no dependencies", task.ID)
			}
			dep := task.DependsOn[0]
			if out, ok := e.Outputs[dep]; ok {
				args[i] = strings.ReplaceAll(arg, "$OUTPUT", out)
			} else {
				return fmt.Errorf("dependency %s output not found", dep)
			}
		} else {
			args[i] = arg
		}
	}

	task.Status = "running"
	task.StartTime = time.Now().UnixMilli()
	eventCh <- Event{Type: EventTaskUpdate, TaskID: task.ID, Status: "running"}

	var err error
	var output []byte

	if task.Type == "tool" {
		cmd, cmdErr := commandForTool(task.Tool, args)
		if cmdErr != nil {
			task.Status = "failed"
			task.Error = cmdErr.Error()
			task.EndTime = time.Now().UnixMilli()
			eventCh <- Event{Type: EventTaskUpdate, TaskID: task.ID, Status: "failed", Error: cmdErr.Error()}
			return cmdErr
		}
		cmd.Stderr = os.Stderr
		output, err = cmd.Output()
	} else {
		if task.Worktree {
			wt, wtErr := worktree.Create()
			if wtErr != nil {
				task.Status = "failed"
				return fmt.Errorf("worktree creation failed: %w", wtErr)
			}
			defer func() {
				if err := worktree.Cleanup(wt); err != nil {
					fmt.Fprintf(os.Stderr, "worktree cleanup warning: %v\n", err)
				}
			}()

			cmd, cmdErr := commandForTool(task.Tool, append(args, "--cwd", wt))
			if cmdErr != nil {
				task.Status = "failed"
				return cmdErr
			}
			cmd.Stderr = os.Stderr
			output, err = cmd.Output()
		} else {
			cmd, cmdErr := commandForTool(task.Tool, args)
			if cmdErr != nil {
				task.Status = "failed"
				return cmdErr
			}
			cmd.Stderr = os.Stderr
			output, err = cmd.Output()
		}
	}

	outputStr := strings.TrimSpace(string(output))

	tokenCount := cost.EstimateTokens(string(output)) + 100
	e.Tokens += tokenCount
	costUSD := cost.EstimateCost(tokenCount)
	e.Cost += costUSD
	task.Cost = costUSD
	task.TokenUsed = tokenCount

	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		task.EndTime = time.Now().UnixMilli()
		eventCh <- Event{Type: EventTaskUpdate, TaskID: task.ID, Status: "failed", Output: outputStr, Error: err.Error()}
		if logErr := e.State.Log(state.JournalEntry{
			SessionID: e.SessionID,
			TaskID:    task.ID,
			Tool:      task.Tool,
			Args:      args,
			Output:    outputStr,
			Error:     err.Error(),
			Status:    "failed",
			Cost:      costUSD,
			Tokens:    tokenCount,
		}); logErr != nil {
			fmt.Fprintf(os.Stderr, "failed to log task failure: %v\n", logErr)
		}
		return err
	}

	task.Status = "completed"
	task.Output = outputStr
	task.EndTime = time.Now().UnixMilli()
	e.Outputs[task.ID] = outputStr
	eventCh <- Event{Type: EventTaskUpdate, TaskID: task.ID, Status: "completed", Output: outputStr}

	if err := e.State.Log(state.JournalEntry{
		SessionID: e.SessionID,
		TaskID:    task.ID,
		Tool:      task.Tool,
		Args:      args,
		Output:    outputStr,
		Status:    "completed",
		Cost:      costUSD,
		Tokens:    tokenCount,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to log task completion: %v\n", err)
	}
	return nil
}

func commandForTool(tool string, args []string) (*exec.Cmd, error) {
	switch tool {
	case "find":
		return exec.Command("find", args...), nil
	case "grep":
		return exec.Command("grep", args...), nil
	case "awk":
		return exec.Command("awk", args...), nil
	case "sed":
		return exec.Command("sed", args...), nil
	case "jq":
		return exec.Command("jq", args...), nil
	case "curl":
		return exec.Command("curl", args...), nil
	case "git":
		return exec.Command("git", args...), nil
	case "docker":
		return exec.Command("docker", args...), nil
	case "kubectl":
		return exec.Command("kubectl", args...), nil
	case "python3":
		return exec.Command("python3", args...), nil
	case "node":
		return exec.Command("node", args...), nil
	case "go":
		return exec.Command("go", args...), nil
	case "ollama":
		return exec.Command("ollama", args...), nil
	case "claude-code":
		return exec.Command("claude-code", args...), nil
	case "gemini-cli":
		return exec.Command("gemini-cli", args...), nil
	default:
		return nil, fmt.Errorf("unsupported executable: %q", tool)
	}
}
