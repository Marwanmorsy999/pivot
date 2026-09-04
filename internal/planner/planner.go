package planner

// TaskType distinguishes CLI tools from AI agents.
type TaskType string

const (
	TypeTool       TaskType = "tool"
	TypeAgent      TaskType = "agent"
	TypeCheckpoint TaskType = "checkpoint"
)

// Task is a single unit of work in a pivot plan.
type Task struct {
	ID          string   `json:"id"          yaml:"id"`
	Type        TaskType `json:"type"        yaml:"type"`
	Tool        string   `json:"tool"        yaml:"tool"`
	Args        []string `json:"args"        yaml:"args"`
	DependsOn   []string `json:"depends_on"  yaml:"depends_on"`
	Description string   `json:"description" yaml:"description"`
	Worktree    bool     `json:"worktree"    yaml:"worktree"`
	// TimeoutSec is the per-task execution timeout in seconds (0 = use default 300s).
	TimeoutSec int `json:"timeout_sec,omitempty" yaml:"timeout_sec,omitempty"`
	// Retries is the number of retry attempts on failure (0 = no retry).
	Retries int `json:"retries,omitempty" yaml:"retries,omitempty"`
	// Before is an optional shell command run before the task executes.
	Before string `json:"before,omitempty" yaml:"before,omitempty"`
	// After is an optional shell command run after the task completes successfully.
	After string `json:"after,omitempty" yaml:"after,omitempty"`
	// Prompt is the message shown to the user for checkpoint tasks.
	Prompt string `json:"prompt,omitempty" yaml:"prompt,omitempty"`
}

// Planner converts a natural-language goal into a structured task graph.
type Planner interface {
	Plan(goal string) ([]Task, error)
}

// WorkflowFile is the top-level structure for YAML workflow files.
type WorkflowFile struct {
	Goal  string `yaml:"goal"`
	Tasks []Task `yaml:"tasks"`
}
