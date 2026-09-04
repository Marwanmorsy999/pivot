package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Marwanmorsy999/pivot/internal/core"
	"github.com/Marwanmorsy999/pivot/internal/planner"
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

// Model is the Bubble Tea model for pivot's TUI.
// All methods use pointer receivers so mutations are preserved across frames.
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
	width              int
	pendingCheckpoint *core.Event
}

// NewModel constructs a TUI model ready to receive events.
func NewModel(sessionID, goal string, tasks []planner.Task, eventCh <-chan core.Event) *Model {
	taskMap := make(map[string]*core.Task, len(tasks))
	taskOrder := make([]string, len(tasks))
	for i, t := range tasks {
		taskMap[t.ID] = &core.Task{Task: t, Status: "pending"}
		taskOrder[i] = t.ID
	}

	cols := []table.Column{
		{Title: "ID", Width: 16},
		{Title: "Type", Width: 6},
		{Title: "Tool", Width: 14},
		{Title: "Description", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Cost", Width: 10},
	}
	rows := make([]table.Row, len(tasks))
	for i, t := range tasks {
		rows[i] = table.Row{
			truncate(t.ID, 16), string(t.Type), truncate(t.Tool, 14),
			truncate(t.Description, 30), "pending", "$0.000000",
		}
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = runningStyle

	tbl := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(min(len(tasks)+1, 12)),
	)
	tbl.SetStyles(table.Styles{
		Header: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")),
		Cell:   lipgloss.NewStyle().Foreground(lipgloss.Color("#EEEEEE")),
	})

	return &Model{
		sessionID: sessionID,
		goal:      goal,
		tasks:     taskMap,
		taskOrder: taskOrder,
		eventCh:   eventCh,
		spinner:   s,
		viewport:  viewport.New(80, 8),
		table:     tbl,
		startTime: time.Now(),
		width:     80,
	}
}

// Init starts the spinner and begins listening for events.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.waitForEvent())
}

// Update handles all incoming messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.viewport.Width = msg.Width
		return m, m.waitForEvent()
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, tea.Batch(cmd, m.waitForEvent())
	case core.Event:
		return m.handleEvent(msg)
	case eventClosedMsg:
		m.done = true
		if m.finalMsg == "" {
			m.finalMsg = "✅ Session complete"
		}
		return m, nil
	case tea.ResumeMsg:
		// TUI restored after checkpoint suspend.
		// The actual prompt happens here, synchronously, before the TUI repaints.
		if m.pendingCheckpoint != nil {
			ev := m.pendingCheckpoint
			m.pendingCheckpoint = nil
			confirmed := promptCheckpoint(ev.TaskID, ev.Prompt)
			ev.RespCh <- core.CheckpointResponse{Confirmed: confirmed}
		}
		return m, m.waitForEvent()
	}
	return m, m.waitForEvent()
}

// View renders the current TUI frame.
func (m *Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("⚡ PIVOT — Hybrid CLI Orchestrator") + "\n")
	fmt.Fprintf(&b, "  Session : %s\n", m.sessionID)
	fmt.Fprintf(&b, "  Goal    : %s\n", truncate(m.goal, m.width-12))
	fmt.Fprintf(&b, "  Runtime : %s\n\n", time.Since(m.startTime).Round(time.Second))

	b.WriteString(titleStyle.Render("📋 Tasks") + "\n")
	b.WriteString(m.table.View() + "\n\n")

	b.WriteString(costStyle.Render(fmt.Sprintf(
		"💰 Cost: $%.6f  |  🔤 Tokens: %d", m.totalCost, m.totalTokens,
	)) + "\n\n")

	b.WriteString(titleStyle.Render("📜 Log") + "\n")
	logs := m.events
	if len(logs) > 8 {
		logs = logs[len(logs)-8:]
	}
	if len(logs) > 0 {
		m.viewport.SetContent(strings.Join(logs, "\n"))
		b.WriteString(m.viewport.View())
	} else {
		b.WriteString(pendingStyle.Render("  Waiting for events..."))
	}
	b.WriteString("\n")

	if m.done {
		b.WriteString("\n")
		if strings.HasPrefix(m.finalMsg, "❌") {
			b.WriteString(failedStyle.Render("  " + m.finalMsg))
		} else {
			b.WriteString(completedStyle.Render("  " + m.finalMsg))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n" + helpStyle.Render("  q / ctrl+c: quit") + "\n")
	return b.String()
}

func (m *Model) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-m.eventCh
		if !ok {
			return eventClosedMsg{}
		}
		return ev
	}
}

func (m *Model) handleEvent(ev core.Event) (tea.Model, tea.Cmd) {
	switch ev.Type {
	case core.EventTaskUpdate:
		task, ok := m.tasks[ev.TaskID]
		if !ok {
			return m, m.waitForEvent()
		}
		task.Status = ev.Status
		if ev.Output != "" {
			task.Output = ev.Output
		}
		if ev.Error != "" {
			task.Error = ev.Error
		}

		annotation := ev.Status
		if ev.Message != "" {
			annotation = fmt.Sprintf("%s (%s)", ev.Status, ev.Message)
		}
		m.events = append(m.events, fmt.Sprintf(
			"[%s] %s %s — %s",
			time.Now().Format("15:04:05"), styleStatus(ev.Status), ev.TaskID, annotation,
		))
		if ev.Error != "" {
			m.events = append(m.events, fmt.Sprintf(
				"         ↳ %s", failedStyle.Render(truncate(ev.Error, 72)),
			))
		}

		rows := m.table.Rows()
		for i, row := range rows {
			if row[0] == truncate(ev.TaskID, 16) {
				costStr := "$0.000000"
				if task.Cost > 0 {
					costStr = fmt.Sprintf("$%.6f", task.Cost)
				}
				rows[i] = table.Row{
					truncate(ev.TaskID, 16), string(task.Type),
					truncate(task.Tool, 14), truncate(task.Description, 30),
					ev.Status, costStr,
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
		m.events = append(m.events, fmt.Sprintf(
			"[%s] %s %s", time.Now().Format("15:04:05"),
			completedStyle.Render("✓"), ev.Message,
		))
		return m, nil

	case core.EventError:
		m.done = true
		m.finalMsg = fmt.Sprintf("❌ Error: %s", ev.Message)
		m.events = append(m.events, fmt.Sprintf(
			"[%s] %s %s", time.Now().Format("15:04:05"),
			failedStyle.Render("✗"), ev.Message,
		))
		return m, nil

	case core.EventCheckpoint:
		m.events = append(m.events, fmt.Sprintf(
			"[%s] ⏸  checkpoint [%s]: %s",
			time.Now().Format("15:04:05"), ev.TaskID, ev.Prompt,
		))
		m.pendingCheckpoint = &ev
		// Suspend releases alt-screen and gives the terminal back.
		return m, tea.Suspend
	}

	return m, m.waitForEvent()
}

// handleSuspendResume is called after tea.Suspend returns control.
// We read the checkpoint answer from stdin and resume the TUI.
func (m *Model) handleSuspendResume(ev core.Event) tea.Cmd {
	return func() tea.Msg {
		confirmed := promptCheckpoint(ev.TaskID, ev.Prompt)
		ev.RespCh <- core.CheckpointResponse{Confirmed: confirmed}
		return tea.ResumeMsg{}
	}
}

// promptCheckpoint reads a y/n answer from stdin after the TUI has suspended.
// Called synchronously from the main goroutine during ResumeMsg handling.
func promptCheckpoint(taskID, prompt string) bool {
	fmt.Fprintf(os.Stderr, "\n⏸  CHECKPOINT [%s]\n   %s [y/n]: ", taskID, prompt)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "y" || answer == "yes" {
			return true
		}
		if answer == "n" || answer == "no" {
			return false
		}
		fmt.Fprint(os.Stderr, "   Please enter y or n: ")
	}
	return false
}

// Run starts the Bubble Tea program with alt-screen mode.
func (m *Model) Run() error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
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

func truncate(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
