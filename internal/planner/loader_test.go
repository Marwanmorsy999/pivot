package planner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Marwanmorsy999/pivot/internal/planner"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "workflow-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoadWorkflowFile_Basic(t *testing.T) {
	yaml := `
goal: test goal
tasks:
  - id: hello
    type: tool
    tool: echo
    args: ["hi"]
    description: say hi
`
	path := writeYAML(t, yaml)
	goal, tasks, err := planner.LoadWorkflowFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if goal != "test goal" {
		t.Errorf("expected goal 'test goal', got %q", goal)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "hello" {
		t.Errorf("expected task id 'hello', got %q", tasks[0].ID)
	}
	if tasks[0].Tool != "echo" {
		t.Errorf("expected tool 'echo', got %q", tasks[0].Tool)
	}
}

func TestLoadWorkflowFile_NoGoal(t *testing.T) {
	yaml := `
tasks:
  - id: t1
    type: tool
    tool: sh
    args: ["-c", "echo ok"]
`
	path := writeYAML(t, yaml)
	goal, tasks, err := planner.LoadWorkflowFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if goal != "" {
		t.Errorf("expected empty goal, got %q", goal)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
}

func TestLoadWorkflowFile_WithHooks(t *testing.T) {
	yaml := `
goal: hooks test
tasks:
  - id: t1
    type: tool
    tool: echo
    args: ["done"]
    before: "echo before"
    after: "echo after"
`
	path := writeYAML(t, yaml)
	_, tasks, err := planner.LoadWorkflowFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tasks[0].Before != "echo before" {
		t.Errorf("expected before hook, got %q", tasks[0].Before)
	}
	if tasks[0].After != "echo after" {
		t.Errorf("expected after hook, got %q", tasks[0].After)
	}
}

func TestLoadWorkflowFile_Checkpoint(t *testing.T) {
	yaml := `
goal: checkpoint test
tasks:
  - id: gate
    type: checkpoint
    prompt: "Ready to continue?"
`
	path := writeYAML(t, yaml)
	_, tasks, err := planner.LoadWorkflowFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tasks[0].Type != planner.TypeCheckpoint {
		t.Errorf("expected checkpoint type, got %q", tasks[0].Type)
	}
	if tasks[0].Prompt != "Ready to continue?" {
		t.Errorf("expected prompt, got %q", tasks[0].Prompt)
	}
}

func TestLoadWorkflowFile_Dependencies(t *testing.T) {
	yaml := `
goal: deps test
tasks:
  - id: a
    type: tool
    tool: echo
    args: ["a"]
  - id: b
    type: tool
    tool: echo
    args: ["b"]
    depends_on: [a]
`
	path := writeYAML(t, yaml)
	_, tasks, err := planner.LoadWorkflowFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks[1].DependsOn) != 1 || tasks[1].DependsOn[0] != "a" {
		t.Errorf("expected depends_on [a], got %v", tasks[1].DependsOn)
	}
}

func TestLoadWorkflowFile_NotFound(t *testing.T) {
	_, _, err := planner.LoadWorkflowFile(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadWorkflowFile_EmptyTasks(t *testing.T) {
	yaml := `goal: empty
tasks: []
`
	path := writeYAML(t, yaml)
	_, _, err := planner.LoadWorkflowFile(path)
	if err == nil {
		t.Error("expected error for empty task list")
	}
}

func TestLoadWorkflowFile_InvalidYAML(t *testing.T) {
	path := writeYAML(t, "this: is: not: valid: yaml: :")
	_, _, err := planner.LoadWorkflowFile(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadWorkflowFile_TimeoutAndRetries(t *testing.T) {
	yaml := `
goal: timeout test
tasks:
  - id: t1
    type: tool
    tool: sh
    args: ["-c", "sleep 1"]
    timeout_sec: 30
    retries: 2
`
	path := writeYAML(t, yaml)
	_, tasks, err := planner.LoadWorkflowFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tasks[0].TimeoutSec != 30 {
		t.Errorf("expected timeout_sec 30, got %d", tasks[0].TimeoutSec)
	}
	if tasks[0].Retries != 2 {
		t.Errorf("expected retries 2, got %d", tasks[0].Retries)
	}
}
