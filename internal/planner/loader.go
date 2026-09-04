package planner

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadWorkflowFile reads a YAML workflow file and returns its tasks.
// The file must contain a top-level "tasks" key; "goal" is optional.
func LoadWorkflowFile(path string) (string, []Task, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is user-supplied CLI argument
	if err != nil {
		return "", nil, fmt.Errorf("read workflow file: %w", err)
	}

	var wf WorkflowFile
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return "", nil, fmt.Errorf("parse workflow file: %w", err)
	}
	if len(wf.Tasks) == 0 {
		return "", nil, fmt.Errorf("workflow file %q contains no tasks", path)
	}
	return wf.Goal, wf.Tasks, nil
}
