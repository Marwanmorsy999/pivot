package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Marwanmorsy999/pivot/internal/core"
	"github.com/Marwanmorsy999/pivot/internal/planner"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginLeft(1)

	pendingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262"))

	runningStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFAB40"))

	completedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#04B575"))

	failedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF4444"))

	skippedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	costStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFD700"))

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		MarginLeft(1)
)

type eventClosedMsg struct{}

type Model struct {
	sessionID   string
	goal        string
	tasks       map[string]*core.Task
	taskOrder   []string
	eventCh     <-chan core.Event
	spinner     spinner.Model
	viewport    viewport.Model
	table       table.Model
	events      []string
	done        bool
	totalCost   float64
	totalTokens int
	startTime   time.Time
	finalMsg    string
}

func NewModel(sessionID, goal string, tasks []planner.Task, eventCh <-chan core.Event) *Model {
	taskMap := make(map[string]*core.Task)
	taskOrder := make([]string, len(tasks))

	for i, t := range tasks {
		ct := &core.Task{Task: t, Status: "pending"}
		taskMap[t.ID] = ct
		taskOrder[i] = t.ID
	}

	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Type", Width: 8},
		{Title: "Tool", Width: 14},
		{Title: "Description", Width: 30},
		{Title: "Status", Width: 12},
		{Title: "Cost", Width: 10},
	}

	rows := make([]table.Row, len(tasks))
	for i, t := range tasks {
		rows[i] = table.Row{
			t.ID,
			string(t.Type),
			t.Tool,
			truncate(t.Description, 28),
			"pending",
			"$0.000000",
		}
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(tasks)+1),
	)

	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		BorderBottom(true)
	ts.Selected = ts.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4"))
	t.SetStyles(ts)

	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	vp := viewport.New(80, 10)

	return &Model{
		sessionID: sessionID,
		goal:      goal,
		tasks:     taskMap,
		taskOrder: taskOrder,
		eventCh:   eventCh,
		spinner:   s,
		viewport:  vp,
		table:     t,
		events:    []string{},
		startTime: time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.waitForEvent(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case core.Event:
		return m.handleEvent(msg)

	case eventClosedMsg:
		m.done = true
		if m.finalMsg == "" {
			m.finalMsg = "✅ Session complete"
		}
		return m, nil
	}

	return m, m.waitForEvent()
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("⚡ PIVOT — Hybrid CLI Orchestrator"))
	b.WriteString("\n")
	_, _ = fmt.Fprintf(&b, "  Session: %s\n", m.sessionID)
	_, _ = fmt.Fprintf(&b, "  Goal: %s\n", m.goal)
	_, _ = fmt.Fprintf(&b, "  Runtime: %s\n", time.Since(m.startTime).Round(time.Second))
	b.WriteString("\n")

	b.WriteString(titleStyle.Render("📋 Tasks"))
	b.WriteString("\n")
	b.WriteString(m.table.View())
	b.WriteString("\n\n")

	_, _ = fmt.Fprintf(&b, "%s\n\n", costStyle.Render(fmt.Sprintf("💰 Cost: $%.6f  |  🔤 Tokens: %d", m.totalCost, m.totalTokens)))

	b.WriteString(titleStyle.Render("📜 Log"))
	b.WriteString("\n")

	if len(m.events) > 0 {
		start := 0
		if len(m.events) > 8 {
			start = len(m.events) - 8
		}
		for _, e := range m.events[start:] {
			_, _ = fmt.Fprintf(&b, "  %s\n", e)
		}
	} else {
		b.WriteString(pendingStyle.Render("  Waiting for events..."))
		b.WriteString("\n")
	}

	if m.done {
		b.WriteString("\n")
		b.WriteString(completedStyle.Render("  " + m.finalMsg))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  q: quit"))
	b.WriteString("\n")

	return b.String()
}

func (m Model) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-m.eventCh
		if !ok {
			return eventClosedMsg{}
		}
		return ev
	}
}

func (m Model) handleEvent(ev core.Event) (tea.Model, tea.Cmd) {
	switch ev.Type {
	case core.EventTaskUpdate:
		task, ok := m.tasks[ev.TaskID]
		if !ok {
			return m, m.waitForEvent()
		}

		task.Status = ev.Status

		statusStr := styleStatus(ev.Status)
		toolStr := task.Tool
		if len(toolStr) > 12 {
			toolStr = toolStr[:12]
		}

		m.events = append(m.events, fmt.Sprintf(
			"[%s] %s %s %s",
			time.Now().Format("15:04:05"),
			statusStr,
			ev.TaskID,
			ev.Status,
		))

		rows := m.table.Rows()
		for i, row := range rows {
			if row[0] == ev.TaskID {
				cost := "$0.000000"
				if task.Cost > 0 {
					cost = fmt.Sprintf("$%.6f", task.Cost)
				}
				rows[i] = table.Row{
					ev.TaskID,
					string(task.Type),
					toolStr,
					truncate(task.Description, 28),
					ev.Status,
					cost,
				}
				break
			}
		}
		m.table.SetRows(rows)

	case core.EventComplete:
		m.done = true
		m.totalCost = ev.Cost
		m.totalTokens = ev.Tokens
		m.finalMsg = ev.Message
		return m, nil

	case core.EventError:
		m.done = true
		m.finalMsg = fmt.Sprintf("❌ Error: %s", ev.Message)
		m.events = append(m.events, fmt.Sprintf(
			"[%s] ❌ %s",
			time.Now().Format("15:04:05"),
			ev.Message,
		))
		return m, nil
	}

	return m, m.waitForEvent()
}

func styleStatus(status string) string {
	switch status {
	case "running":
		return runningStyle.Render("▶")
	case "completed":
		return completedStyle.Render("✓")
	case "failed":
		return failedStyle.Render("✗")
	case "skipped":
		return skippedStyle.Render("⊘")
	default:
		return pendingStyle.Render("○")
	}
}

func (m *Model) Run() error {
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
