package core

import (
	"pivot/internal/planner"
	"reflect"
	"testing"
)

func TestGraphOrder_Linear(t *testing.T) {
	tasks := []*Task{
		{Task: planner.Task{ID: "a", DependsOn: []string{}}},
		{Task: planner.Task{ID: "b", DependsOn: []string{"a"}}},
		{Task: planner.Task{ID: "c", DependsOn: []string{"b"}}},
	}
	g := NewGraph(tasks)
	order, err := g.Order()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("expected %v, got %v", expected, order)
	}
}

func TestGraphOrder_Diamond(t *testing.T) {
	tasks := []*Task{
		{Task: planner.Task{ID: "a", DependsOn: []string{}}},
		{Task: planner.Task{ID: "b", DependsOn: []string{"a"}}},
		{Task: planner.Task{ID: "c", DependsOn: []string{"a"}}},
		{Task: planner.Task{ID: "d", DependsOn: []string{"b", "c"}}},
	}
	g := NewGraph(tasks)
	order, err := g.Order()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(order))
	}
	if order[0] != "a" || order[3] != "d" {
		t.Errorf("expected a first and d last, got %v", order)
	}
}

func TestGraphOrder_Cycle(t *testing.T) {
	tasks := []*Task{
		{Task: planner.Task{ID: "a", DependsOn: []string{"b"}}},
		{Task: planner.Task{ID: "b", DependsOn: []string{"a"}}},
	}
	g := NewGraph(tasks)
	_, err := g.Order()
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestGraphOrder_MissingDependency(t *testing.T) {
	tasks := []*Task{
		{Task: planner.Task{ID: "a", DependsOn: []string{"x"}}},
	}
	g := NewGraph(tasks)
	_, err := g.Order()
	if err == nil {
		t.Fatal("expected missing dependency error")
	}
}

func TestGraphOrder_Empty(t *testing.T) {
	g := NewGraph([]*Task{})
	order, err := g.Order()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 0 {
		t.Errorf("expected empty order, got %v", order)
	}
}

func TestGraphGetTask(t *testing.T) {
	tasks := []*Task{
		{Task: planner.Task{ID: "a", Description: "test"}},
	}
	g := NewGraph(tasks)
	task := g.GetTask("a")
	if task == nil {
		t.Fatal("expected task, got nil")
	}
	if task.Description != "test" {
		t.Errorf("expected 'test', got %q", task.Description)
	}
	if g.GetTask("missing") != nil {
		t.Error("expected nil for missing task")
	}
}
