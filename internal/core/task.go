package core

import "pivot/internal/planner"

type Task struct {
	planner.Task
	Status    string  `json:"status"`
	Output    string  `json:"output"`
	Error     string  `json:"error"`
	Cost      float64 `json:"cost"`
	TokenUsed int     `json:"token_used"`
	StartTime int64   `json:"start_time"`
	EndTime   int64   `json:"end_time"`
}
