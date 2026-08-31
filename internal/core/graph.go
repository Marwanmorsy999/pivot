package core

import "fmt"

type Graph struct {
	tasks map[string]*Task
}

func NewGraph(tasks []*Task) *Graph {
	g := &Graph{tasks: make(map[string]*Task)}
	for _, t := range tasks {
		g.tasks[t.ID] = t
	}
	return g
}

func (g *Graph) Order() ([]string, error) {
	inDegree := make(map[string]int)
	adj := make(map[string][]string)

	for id := range g.tasks {
		inDegree[id] = 0
		adj[id] = []string{}
	}

	for id, task := range g.tasks {
		for _, dep := range task.DependsOn {
			if _, ok := g.tasks[dep]; !ok {
				return nil, fmt.Errorf("dependency %s not found", dep)
			}
			inDegree[id]++
			adj[dep] = append(adj[dep], id)
		}
	}

	queue := []string{}
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var result []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, next := range adj[current] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(result) != len(g.tasks) {
		return nil, fmt.Errorf("cycle detected")
	}
	return result, nil
}

func (g *Graph) GetTask(id string) *Task {
	return g.tasks[id]
}
