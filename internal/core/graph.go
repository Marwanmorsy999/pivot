package core

import (
	"fmt"
	"sort"
)

// Graph holds the dependency graph for a task set.
type Graph struct {
	tasks map[string]*Task
}

// NewGraph builds a Graph from a slice of tasks.
func NewGraph(tasks []*Task) *Graph {
	g := &Graph{tasks: make(map[string]*Task, len(tasks))}
	for _, t := range tasks {
		g.tasks[t.ID] = t
	}
	return g
}

// Order returns task IDs in topological order (Kahn's algorithm).
// Returns an error if a cycle is detected or a dependency is missing.
func (g *Graph) Order() ([]string, error) {
	inDegree := make(map[string]int, len(g.tasks))
	adj := make(map[string][]string, len(g.tasks))

	for id := range g.tasks {
		inDegree[id] = 0
		adj[id] = []string{}
	}

	for id, task := range g.tasks {
		for _, dep := range task.DependsOn {
			if _, ok := g.tasks[dep]; !ok {
				return nil, fmt.Errorf("dependency %q of task %q not found", dep, id)
			}
			inDegree[id]++
			adj[dep] = append(adj[dep], id)
		}
	}

	// Seed queue with zero-in-degree nodes, sorted for determinism.
	queue := make([]string, 0, len(g.tasks))
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	result := make([]string, 0, len(g.tasks))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		neighbors := adj[current]
		sort.Strings(neighbors)
		for _, next := range neighbors {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(result) != len(g.tasks) {
		return nil, fmt.Errorf("cycle detected in task dependency graph")
	}
	return result, nil
}

// Waves returns tasks grouped by execution wave — tasks in the same wave
// have no dependencies on each other and can run in parallel.
func (g *Graph) Waves() ([][]string, error) {
	order, err := g.Order()
	if err != nil {
		return nil, err
	}

	// Compute depth of each node (longest path from any root).
	depth := make(map[string]int, len(g.tasks))
	for _, id := range order {
		task := g.tasks[id]
		for _, dep := range task.DependsOn {
			if depth[dep]+1 > depth[id] {
				depth[id] = depth[dep] + 1
			}
		}
	}

	// Group by depth.
	maxDepth := 0
	for _, d := range depth {
		if d > maxDepth {
			maxDepth = d
		}
	}
	waves := make([][]string, maxDepth+1)
	for _, id := range order {
		d := depth[id]
		waves[d] = append(waves[d], id)
	}
	return waves, nil
}

// GetTask returns the Task for the given ID.
func (g *Graph) GetTask(id string) *Task { return g.tasks[id] }
