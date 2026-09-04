package planner_test

import (
	"strings"
	"testing"

	"github.com/Marwanmorsy999/pivot/internal/planner"
)

func TestValidate_HappyPath(t *testing.T) {
	tasks := []planner.Task{
		{ID: "fetch", Type: planner.TypeTool, Tool: "curl", Args: []string{"https://example.com"}},
		{ID: "parse", Type: planner.TypeTool, Tool: "jq", Args: []string{".[0]", "$OUTPUT[fetch]"}, DependsOn: []string{"fetch"}},
	}
	if err := planner.Validate(tasks); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_EmptyPlan(t *testing.T) {
	if err := planner.Validate(nil); err == nil {
		t.Fatal("expected error for empty plan")
	}
}

func TestValidate_EmptyID(t *testing.T) {
	tasks := []planner.Task{{ID: "", Type: planner.TypeTool, Tool: "curl"}}
	err := planner.Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "empty id") {
		t.Fatalf("expected empty id error, got: %v", err)
	}
}

func TestValidate_DuplicateID(t *testing.T) {
	tasks := []planner.Task{
		{ID: "a", Type: planner.TypeTool, Tool: "curl"},
		{ID: "a", Type: planner.TypeTool, Tool: "jq"},
	}
	err := planner.Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate id error, got: %v", err)
	}
}

func TestValidate_InvalidType(t *testing.T) {
	tasks := []planner.Task{{ID: "x", Type: "wizard", Tool: "curl"}}
	err := planner.Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "invalid type") {
		t.Fatalf("expected invalid type error, got: %v", err)
	}
}

func TestValidate_UnsupportedTool(t *testing.T) {
	tasks := []planner.Task{{ID: "x", Type: planner.TypeTool, Tool: "notarealtool"}}
	err := planner.Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "unsupported tool") {
		t.Fatalf("expected unsupported tool error, got: %v", err)
	}
}

func TestValidate_EmptyTool(t *testing.T) {
	tasks := []planner.Task{{ID: "x", Type: planner.TypeTool, Tool: ""}}
	err := planner.Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "tool is empty") {
		t.Fatalf("expected empty tool error, got: %v", err)
	}
}

func TestValidate_MissingDependency(t *testing.T) {
	tasks := []planner.Task{
		{ID: "a", Type: planner.TypeTool, Tool: "curl", DependsOn: []string{"ghost"}},
	}
	err := planner.Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "unknown task") {
		t.Fatalf("expected unknown dep error, got: %v", err)
	}
}

func TestValidate_AgentType(t *testing.T) {
	tasks := []planner.Task{
		{ID: "think", Type: planner.TypeAgent, Tool: "claude-code", Args: []string{"fix this"}},
	}
	if err := planner.Validate(tasks); err != nil {
		t.Fatalf("agent task should be valid, got: %v", err)
	}
}

func TestValidate_ShellPassthrough(t *testing.T) {
	tasks := []planner.Task{
		{ID: "run", Type: planner.TypeTool, Tool: "bash", Args: []string{"-c", "echo hello"}},
	}
	if err := planner.Validate(tasks); err != nil {
		t.Fatalf("bash should be allowed, got: %v", err)
	}
}

func TestValidate_DiamondDependency(t *testing.T) {
	tasks := []planner.Task{
		{ID: "a", Type: planner.TypeTool, Tool: "echo"},
		{ID: "b", Type: planner.TypeTool, Tool: "echo", DependsOn: []string{"a"}},
		{ID: "c", Type: planner.TypeTool, Tool: "echo", DependsOn: []string{"a"}},
		{ID: "d", Type: planner.TypeTool, Tool: "echo", DependsOn: []string{"b", "c"}},
	}
	if err := planner.Validate(tasks); err != nil {
		t.Fatalf("diamond dep should be valid, got: %v", err)
	}
}

func TestParseTasks_PlainJSON(t *testing.T) {
	raw := `{"tasks":[{"id":"a","type":"tool","tool":"echo","args":[],"depends_on":[]}]}`
	tasks, err := planner.ParseTasks(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ID != "a" {
		t.Fatalf("unexpected tasks: %v", tasks)
	}
}

func TestParseTasks_StripsFences(t *testing.T) {
	raw := "```json\n{\"tasks\":[]}\n```"
	tasks, err := planner.ParseTasks(raw)
	if err != nil {
		t.Fatalf("ParseTasks failed: %v", err)
	}
	if tasks == nil {
		t.Fatal("expected non-nil tasks slice")
	}
}

func TestValidate_CheckpointNoTool(t *testing.T) {
	tasks := []Task{
		{ID: "gate", Type: TypeCheckpoint, Prompt: "Ready?"},
	}
	if err := Validate(tasks); err != nil {
		t.Errorf("checkpoint without tool should be valid, got: %v", err)
	}
}

func TestValidate_CheckpointWithDep(t *testing.T) {
	tasks := []Task{
		{ID: "build", Type: TypeTool, Tool: "sh", Args: []string{"-c", "make"}},
		{ID: "gate", Type: TypeCheckpoint, DependsOn: []string{"build"}, Prompt: "Deploy?"},
	}
	if err := Validate(tasks); err != nil {
		t.Errorf("checkpoint with valid dep should pass, got: %v", err)
	}
}

func TestValidate_InvalidType(t *testing.T) {
	tasks := []Task{
		{ID: "t1", Type: "plugin", Tool: "sh"},
	}
	if err := Validate(tasks); err == nil {
		t.Error("expected error for invalid type 'plugin'")
	}
}

