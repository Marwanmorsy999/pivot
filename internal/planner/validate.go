package planner

import (
	"fmt"
	"strings"
)

// validTools is the set of permitted executables for tool-type tasks.
// Must stay in sync with allowedTools in internal/core/executor.go.
var validTools = map[string]bool{
	// Unix core
	"find": true, "grep": true, "awk": true, "sed": true, "cat": true,
	"echo": true, "wc": true, "sort": true, "uniq": true, "head": true,
	"tail": true, "xargs": true, "tar": true, "zip": true, "unzip": true,
	"cut": true, "tr": true, "tee": true, "diff": true, "patch": true,
	"ls": true, "cp": true, "mv": true, "rm": true, "mkdir": true,
	"chmod": true, "chown": true, "touch": true, "stat": true, "file": true,
	"env": true, "printenv": true, "which": true, "date": true, "sleep": true,
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

// validTypes is the set of permitted task type values.
var validTypes = map[TaskType]bool{
	TypeTool:       true,
	TypeAgent:      true,
	TypeCheckpoint: true,
}

// Validate checks a task plan for structural correctness before execution.
// It verifies: non-empty plan, unique IDs, valid types, permitted tools,
// and dependency graph integrity (no references to undefined task IDs).
func Validate(tasks []Task) error {
	if len(tasks) == 0 {
		return fmt.Errorf("task plan is empty")
	}

	seen := make(map[string]bool, len(tasks))
	for i, t := range tasks {
		if t.ID == "" {
			return fmt.Errorf("task[%d] has empty id", i)
		}
		if seen[t.ID] {
			return fmt.Errorf("duplicate task id %q at index %d", t.ID, i)
		}
		seen[t.ID] = true

		if !validTypes[t.Type] {
			return fmt.Errorf("task %q: invalid type %q (must be 'tool', 'agent', or 'checkpoint')", t.ID, t.Type)
		}

		// Checkpoint tasks need no tool.
		if t.Type == TypeCheckpoint {
			continue
		}

		if t.Tool == "" {
			return fmt.Errorf("task %q: tool is empty", t.ID)
		}
		// Allow "sh -c" style — check only the executable name.
		toolName := strings.SplitN(t.Tool, " ", 2)[0]
		if !validTools[toolName] {
			return fmt.Errorf("task %q: unsupported tool %q", t.ID, toolName)
		}
	}

	// Dependency existence check (second pass, after all IDs are known).
	for _, t := range tasks {
		for _, dep := range t.DependsOn {
			if !seen[dep] {
				return fmt.Errorf("task %q depends on unknown task %q", t.ID, dep)
			}
		}
	}

	return nil
}
