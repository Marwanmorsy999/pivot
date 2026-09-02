package planner

// TaskType distinguishes CLI tools from AI agents.
type TaskType string

const (
	TypeTool  TaskType = "tool"
	TypeAgent TaskType = "agent"
)

// Task is a single unit of work in a pivot plan.
type Task struct {
	ID          string   `json:"id"`
	Type        TaskType `json:"type"`
	Tool        string   `json:"tool"`
	Args        []string `json:"args"`
	DependsOn   []string `json:"depends_on"`
	Description string   `json:"description"`
	Worktree    bool     `json:"worktree"`
	// TimeoutSec is the per-task execution timeout in seconds (0 = use default 300s).
	TimeoutSec int `json:"timeout_sec,omitempty"`
	// Retries is the number of retry attempts on failure (0 = no retry).
	Retries int `json:"retries,omitempty"`
}

// Planner converts a natural-language goal into a structured task graph.
type Planner interface {
	Plan(goal string) ([]Task, error)
}
