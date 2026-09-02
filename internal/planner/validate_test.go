package planner

import (
	"strings"
	"testing"
)

func TestValidate_HappyPath(t *testing.T) {
	tasks := []Task{
		{ID: "fetch", Type: TypeTool, Tool: "curl", Args: []string{"https://example.com"}},
		{ID: "parse", Type: TypeTool, Tool: "jq", Args: []string{".[0]", "$OUTPUT[fetch]"}, DependsOn: []string{"fetch"}},
	}
	if err := Validate(tasks); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_EmptyPlan(t *testing.T) {
	if err := Validate(nil); err == nil {
		t.Fatal("expected error for empty plan")
	}
}

func TestValidate_EmptyID(t *testing.T) {
	tasks := []Task{{ID: "", Type: TypeTool, Tool: "curl"}}
	err := Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "empty id") {
		t.Fatalf("expected empty id error, got: %v", err)
	}
}

func TestValidate_DuplicateID(t *testing.T) {
	tasks := []Task{
		{ID: "a", Type: TypeTool, Tool: "curl"},
		{ID: "a", Type: TypeTool, Tool: "jq"},
	}
	err := Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate id error, got: %v", err)
	}
}

func TestValidate_InvalidType(t *testing.T) {
	tasks := []Task{{ID: "x", Type: "wizard", Tool: "curl"}}
	err := Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "invalid type") {
		t.Fatalf("expected invalid type error, got: %v", err)
	}
}

func TestValidate_UnsupportedTool(t *testing.T) {
	tasks := []Task{{ID: "x", Type: TypeTool, Tool: "notarealtool"}}
	err := Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "unsupported tool") {
		t.Fatalf("expected unsupported tool error, got: %v", err)
	}
}

func TestValidate_EmptyTool(t *testing.T) {
	tasks := []Task{{ID: "x", Type: TypeTool, Tool: ""}}
	err := Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "tool is empty") {
		t.Fatalf("expected empty tool error, got: %v", err)
	}
}

func TestValidate_MissingDependency(t *testing.T) {
	tasks := []Task{
		{ID: "a", Type: TypeTool, Tool: "curl", DependsOn: []string{"ghost"}},
	}
	err := Validate(tasks)
	if err == nil || !strings.Contains(err.Error(), "unknown task") {
		t.Fatalf("expected unknown dep error, got: %v", err)
	}
}

func TestValidate_AgentType(t *testing.T) {
	tasks := []Task{
		{ID: "think", Type: TypeAgent, Tool: "claude-code", Args: []string{"fix this bug"}},
	}
	if err := Validate(tasks); err != nil {
		t.Fatalf("agent task should be valid, got: %v", err)
	}
}

func TestValidate_ShellPassthrough(t *testing.T) {
	tasks := []Task{
		{ID: "run", Type: TypeTool, Tool: "bash", Args: []string{"-c", "echo hello"}},
	}
	if err := Validate(tasks); err != nil {
		t.Fatalf("bash should be allowed, got: %v", err)
	}
}

func TestValidate_DiamondDependency(t *testing.T) {
	// A -> B, A -> C, B -> D, C -> D (diamond)
	tasks := []Task{
		{ID: "a", Type: TypeTool, Tool: "echo"},
		{ID: "b", Type: TypeTool, Tool: "echo", DependsOn: []string{"a"}},
		{ID: "c", Type: TypeTool, Tool: "echo", DependsOn: []string{"a"}},
		{ID: "d", Type: TypeTool, Tool: "echo", DependsOn: []string{"b", "c"}},
	}
	if err := Validate(tasks); err != nil {
		t.Fatalf("diamond dep should be valid, got: %v", err)
	}
}

func TestParseTasks_StripsFences(t *testing.T) {
	raw := "```json\n{\"tasks\":[]}\n```"
	tasks, err := parseTasks(raw)
	if err != nil {
		t.Fatalf("parseTasks failed: %v", err)
	}
	if tasks == nil {
		t.Fatal("expected non-nil tasks slice")
	}
}

func TestParseTasks_PlainJSON(t *testing.T) {
	raw := `{"tasks":[{"id":"a","type":"tool","tool":"echo","args":[],"depends_on":[]}]}`
	tasks, err := parseTasks(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ID != "a" {
		t.Fatalf("unexpected tasks: %v", tasks)
	}
}
