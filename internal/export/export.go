// Package export renders pivot sessions as human-readable reports.
package export

import (
	"fmt"
	"strings"
	"time"
)

// TaskRecord holds the data needed to render one task in a report.
type TaskRecord struct {
	ID          string
	Type        string
	Tool        string
	Description string
	Status      string
	Output      string
	Error       string
	Cost        float64
	Tokens      int
	StartMS     int64
	EndMS       int64
}

// Report renders a session as a Markdown document.
func Report(sessionID, goal string, tasks []TaskRecord, totalCost float64, totalTokens int) string {
	var b strings.Builder

	now := time.Now().Format("2006-01-02 15:04:05")
	b.WriteString("# Pivot Session Report\n\n")
	fmt.Fprintf(&b, "**Session:** `%s`\n", sessionID)
	fmt.Fprintf(&b, "**Goal:** %s\n", goal)
	fmt.Fprintf(&b, "**Generated:** %s\n\n", now)
	fmt.Fprintf(&b, "**Total Cost:** $%.6f | **Total Tokens:** %d\n\n", totalCost, totalTokens)

	// Summary table
	b.WriteString("## Task Summary\n\n")
	b.WriteString("| # | ID | Type | Tool | Status | Cost | Duration |\n")
	b.WriteString("|---|----|----|------|--------|------|----------|\n")
	for i, t := range tasks {
		dur := "-"
		if t.StartMS > 0 && t.EndMS > 0 {
			ms := t.EndMS - t.StartMS
			dur = fmt.Sprintf("%dms", ms)
			if ms > 1000 {
				dur = fmt.Sprintf("%.1fs", float64(ms)/1000)
			}
		}
		costStr := "-"
		if t.Cost > 0 {
			costStr = fmt.Sprintf("$%.6f", t.Cost)
		}
		icon := statusIcon(t.Status)
		fmt.Fprintf(&b, "| %d | `%s` | %s | `%s` | %s %s | %s | %s |\n",
			i+1, t.ID, t.Type, t.Tool, icon, t.Status, costStr, dur)
	}
	b.WriteString("\n")

	// Task details
	b.WriteString("## Task Details\n\n")
	for i, t := range tasks {
		fmt.Fprintf(&b, "### %d. %s (`%s`)\n\n", i+1, t.ID, t.Status)
		if t.Description != "" {
			fmt.Fprintf(&b, "**Description:** %s\n\n", t.Description)
		}
		fmt.Fprintf(&b, "**Type:** %s | **Tool:** `%s`\n\n", t.Type, t.Tool)
		if t.Output != "" {
			b.WriteString("**Output:**\n\n```\n")
			out := t.Output
			if len(out) > 2000 {
				out = out[:2000] + "\n... (truncated)"
			}
			b.WriteString(out)
			b.WriteString("\n```\n\n")
		}
		if t.Error != "" {
			fmt.Fprintf(&b, "**Error:** `%s`\n\n", t.Error)
		}
	}

	return b.String()
}

func statusIcon(status string) string {
	switch status {
	case "completed":
		return "✅"
	case "failed":
		return "❌"
	case "skipped":
		return "⊘"
	default:
		return "○"
	}
}
