package planner

import (
	"fmt"
	"strings"
)

// validTools is the set of permitted executables for tool-type tasks.
var validTools = map[string]bool{
	// Unix core
	"find": true, "grep": true, "awk": true, "sed": true, "cat": true,
	"echo": true, "wc": true, "sort": true, "uniq": true, "head": true,
	"tail": true, "xargs": true, "tar": true, "zip": true, "unzip": true,
	"cut": true, "tr": true, "tee": true, "diff": true, "patch": true,
	// Network / data
	"jq": true, "curl": true, "wget": true, "ssh": true, "rsync": true,
	// Dev tools
	"git": true, "docker": true, "kubectl": true, "make": true,
	"python3": true, "node": true, "go": true,
	"npm": true, "pip": true, "cargo": true,
	"terraform": true, "helm": true,
	// Cloud CLIs
	"aws": true, "gcloud": true, "az": true,
	// Shell passthrough
	"sh": true, "bash": true,
	// AI agents
	"ollama": true, "claude-code": true, "gemini-cli": true,
}

// validTypes is the set of permitted task type values.
var validTypes = map[TaskType]bool{
	TypeTool:  true,
	TypeAgent: true,
}

// Validate checks a task plan for structural correctness before execution.
// It verifies: unique IDs, valid type values, permitted tools, dependency graph integrity.
func Validate(tasks []Task) error {
	if len(tasks) == 0 {
		return fmt.Errorf("task plan is empty")
	}

	seen := make(map[string]bool, len(tasks))
	for i, t := range tasks {
		// ID uniqueness
		if t.ID == "" {
			return fmt.Errorf("task[%d] has empty id", i)
		}
		if seen[t.ID] {
			return fmt.Errorf("duplicate task id %q at index %d", t.ID, i)
		}
		seen[t.ID] = true

		// Type check
		if !validTypes[t.Type] {
			return fmt.Errorf("task %q: invalid type %q (must be 'tool' or 'agent')", t.ID, t.Type)
		}

		// Tool check
		if t.Tool == "" {
			return fmt.Errorf("task %q: tool is empty", t.ID)
		}
		toolName := strings.SplitN(t.Tool, " ", 2)[0] // handle "sh -c" style
		if !validTools[toolName] {
			return fmt.Errorf("task %q: unsupported tool %q", t.ID, toolName)
		}
	}

	// Dependency existence check
	for _, t := range tasks {
		for _, dep := range t.DependsOn {
			if !seen[dep] {
				return fmt.Errorf("task %q depends on unknown task %q", t.ID, dep)
			}
		}
	}

	return nil
}
