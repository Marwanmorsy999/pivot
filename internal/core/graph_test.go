package core

import (
	"testing"

	"github.com/Marwanmorsy999/pivot/internal/planner"
)

func makeTasks(specs []struct {
	id   string
	deps []string
}) []*Task {
	tasks := make([]*Task, len(specs))
	for i, s := range specs {
		tasks[i] = &Task{
			Task:   planner.Task{ID: s.id, DependsOn: s.deps, Type: planner.TypeTool, Tool: "echo"},
			Status: "pending",
		}
	}
	return tasks
}

func TestGraph_OrderLinear(t *testing.T) {
	tasks := makeTasks([]struct {
		id   string
		deps []string
	}{
		{"a", nil},
		{"b", []string{"a"}},
		{"c", []string{"b"}},
	})
	g := NewGraph(tasks)
	order, err := g.Order()
	if err != nil {
		t.Fatal(err)
	}
	if order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Fatalf("unexpected order: %v", order)
	}
}

func TestGraph_OrderParallel(t *testing.T) {
	// a has no deps; b and c both depend on a only; independent of each other
	tasks := makeTasks([]struct {
		id   string
		deps []string
	}{
		{"a", nil},
		{"b", []string{"a"}},
		{"c", []string{"a"}},
	})
	g := NewGraph(tasks)
	order, err := g.Order()
	if err != nil {
		t.Fatal(err)
	}
	if order[0] != "a" {
		t.Fatalf("a must be first, got %v", order)
	}
	// b and c can be in any order after a
	if len(order) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(order))
	}
}

func TestGraph_CycleDetection(t *testing.T) {
	tasks := makeTasks([]struct {
		id   string
		deps []string
	}{
		{"a", []string{"b"}},
		{"b", []string{"a"}},
	})
	g := NewGraph(tasks)
	_, err := g.Order()
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestGraph_MissingDependency(t *testing.T) {
	tasks := makeTasks([]struct {
		id   string
		deps []string
	}{
		{"a", []string{"ghost"}},
	})
	g := NewGraph(tasks)
	_, err := g.Order()
	if err == nil {
		t.Fatal("expected missing dep error")
	}
}

func TestGraph_SingleNode(t *testing.T) {
	tasks := makeTasks([]struct {
		id   string
		deps []string
	}{
		{"solo", nil},
	})
	g := NewGraph(tasks)
	order, err := g.Order()
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 1 || order[0] != "solo" {
		t.Fatalf("unexpected order: %v", order)
	}
}

func TestGraph_Diamond(t *testing.T) {
	// A -> B, A -> C, B -> D, C -> D
	tasks := makeTasks([]struct {
		id   string
		deps []string
	}{
		{"A", nil},
		{"B", []string{"A"}},
		{"C", []string{"A"}},
		{"D", []string{"B", "C"}},
	})
	g := NewGraph(tasks)
	order, err := g.Order()
	if err != nil {
		t.Fatal(err)
	}
	// A must be first, D must be last
	if order[0] != "A" {
		t.Fatalf("A must be first, got %v", order)
	}
	if order[len(order)-1] != "D" {
		t.Fatalf("D must be last, got %v", order)
	}
}

func TestGraph_WavesLinear(t *testing.T) {
	tasks := makeTasks([]struct {
		id   string
		deps []string
	}{
		{"a", nil},
		{"b", []string{"a"}},
		{"c", []string{"b"}},
	})
	g := NewGraph(tasks)
	waves, err := g.Waves()
	if err != nil {
		t.Fatal(err)
	}
	if len(waves) != 3 {
		t.Fatalf("expected 3 waves for linear chain, got %d: %v", len(waves), waves)
	}
}

func TestGraph_WavesParallel(t *testing.T) {
	// a -> {b, c, d} — b, c, d are independent and should be in the same wave
	tasks := makeTasks([]struct {
		id   string
		deps []string
	}{
		{"a", nil},
		{"b", []string{"a"}},
		{"c", []string{"a"}},
		{"d", []string{"a"}},
	})
	g := NewGraph(tasks)
	waves, err := g.Waves()
	if err != nil {
		t.Fatal(err)
	}
	if len(waves) != 2 {
		t.Fatalf("expected 2 waves, got %d: %v", len(waves), waves)
	}
	if len(waves[0]) != 1 || waves[0][0] != "a" {
		t.Fatalf("wave 0 should be [a], got %v", waves[0])
	}
	if len(waves[1]) != 3 {
		t.Fatalf("wave 1 should have 3 tasks, got %v", waves[1])
	}
}

func TestGraph_WavesDiamond(t *testing.T) {
	tasks := makeTasks([]struct {
		id   string
		deps []string
	}{
		{"A", nil},
		{"B", []string{"A"}},
		{"C", []string{"A"}},
		{"D", []string{"B", "C"}},
	})
	g := NewGraph(tasks)
	waves, err := g.Waves()
	if err != nil {
		t.Fatal(err)
	}
	// wave 0: A, wave 1: B+C, wave 2: D
	if len(waves) != 3 {
		t.Fatalf("expected 3 waves for diamond, got %d: %v", len(waves), waves)
	}
	if len(waves[1]) != 2 {
		t.Fatalf("wave 1 should have B and C, got %v", waves[1])
	}
}
