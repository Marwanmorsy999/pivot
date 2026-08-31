package planner

type TaskType string

const (
	TypeTool  TaskType = "tool"
	TypeAgent TaskType = "agent"
)

type Task struct {
	ID          string   `json:"id"`
	Type        TaskType `json:"type"`
	Tool        string   `json:"tool"`
	Args        []string `json:"args"`
	DependsOn   []string `json:"depends_on"`
	Description string   `json:"description"`
	Worktree    bool     `json:"worktree"`
}

type Planner interface {
	Plan(goal string) ([]Task, error)
}
