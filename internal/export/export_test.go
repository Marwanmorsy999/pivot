package export_test

import (
	"strings"
	"testing"

	"github.com/Marwanmorsy999/pivot/internal/export"
)

func TestReportContainsSessionID(t *testing.T) {
	got := export.Report("sess_abc123", "test goal", nil, 0, 0)
	if !strings.Contains(got, "sess_abc123") {
		t.Error("report missing session ID")
	}
}

func TestReportContainsGoal(t *testing.T) {
	got := export.Report("s1", "deploy to production", nil, 0, 0)
	if !strings.Contains(got, "deploy to production") {
		t.Error("report missing goal")
	}
}

func TestReportCostFormatting(t *testing.T) {
	got := export.Report("s1", "g", nil, 0.001234, 500)
	if !strings.Contains(got, "$0.001234") {
		t.Errorf("expected formatted cost in report, got:\n%s", got)
	}
	if !strings.Contains(got, "500") {
		t.Error("expected token count in report")
	}
}

func TestReportTaskSummaryTable(t *testing.T) {
	tasks := []export.TaskRecord{
		{ID: "fetch", Type: "tool", Tool: "curl", Status: "completed", Cost: 0.0001, StartMS: 1000, EndMS: 2500},
		{ID: "parse", Type: "tool", Tool: "jq",   Status: "failed",    Error: "exit 1"},
	}
	got := export.Report("s1", "g", tasks, 0.0001, 100)

	if !strings.Contains(got, "fetch") {
		t.Error("missing task ID fetch")
	}
	if !strings.Contains(got, "parse") {
		t.Error("missing task ID parse")
	}
	if !strings.Contains(got, "✅") {
		t.Error("missing completed icon")
	}
	if !strings.Contains(got, "❌") {
		t.Error("missing failed icon")
	}
	if !strings.Contains(got, "1.5s") {
		t.Error("expected 1.5s duration for 1500ms task")
	}
}

func TestReportTaskOutput(t *testing.T) {
	tasks := []export.TaskRecord{
		{ID: "t1", Type: "tool", Tool: "echo", Status: "completed", Output: "hello world"},
	}
	got := export.Report("s1", "g", tasks, 0, 0)
	if !strings.Contains(got, "hello world") {
		t.Error("report missing task output")
	}
}

func TestReportOutputTruncation(t *testing.T) {
	bigOutput := strings.Repeat("x", 3000)
	tasks := []export.TaskRecord{
		{ID: "t1", Type: "tool", Tool: "cat", Status: "completed", Output: bigOutput},
	}
	got := export.Report("s1", "g", tasks, 0, 0)
	if !strings.Contains(got, "truncated") {
		t.Error("expected truncation marker for large output")
	}
}

func TestReportErrorField(t *testing.T) {
	tasks := []export.TaskRecord{
		{ID: "t1", Type: "tool", Tool: "sh", Status: "failed", Error: "exit status 127"},
	}
	got := export.Report("s1", "g", tasks, 0, 0)
	if !strings.Contains(got, "exit status 127") {
		t.Error("report missing error message")
	}
}

func TestReportDescriptionField(t *testing.T) {
	tasks := []export.TaskRecord{
		{ID: "t1", Type: "tool", Tool: "git", Status: "completed", Description: "Clone the repo"},
	}
	got := export.Report("s1", "g", tasks, 0, 0)
	if !strings.Contains(got, "Clone the repo") {
		t.Error("report missing description")
	}
}

func TestReportDurationMs(t *testing.T) {
	tasks := []export.TaskRecord{
		{ID: "t1", Type: "tool", Tool: "echo", Status: "completed", StartMS: 1000, EndMS: 1800},
	}
	got := export.Report("s1", "g", tasks, 0, 0)
	if !strings.Contains(got, "800ms") {
		t.Errorf("expected 800ms duration, got:\n%s", got)
	}
}

func TestReportEmptyTasks(t *testing.T) {
	got := export.Report("s1", "empty run", nil, 0, 0)
	if !strings.Contains(got, "# Pivot Session Report") {
		t.Error("report missing header")
	}
}
