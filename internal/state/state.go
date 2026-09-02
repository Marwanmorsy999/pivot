package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Marwanmorsy999/pivot/internal/paths"
	_ "github.com/mattn/go-sqlite3"
)

// JournalEntry represents a single logged task execution.
type JournalEntry struct {
	SessionID string
	TaskID    string
	Tool      string
	Args      []string
	Output    string
	Error     string
	Status    string
	Cost      float64
	Tokens    int
}

// State manages persistent session and task data in SQLite.
type State struct {
	db *sql.DB
}

func New() (*State, error) {
	stateFile, err := paths.StateFile()
	if err != nil {
		return nil, fmt.Errorf("resolve state path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(stateFile), 0700); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}
	db, err := sql.Open("sqlite3", stateFile)
	if err != nil {
		return nil, err
	}
	s := &State{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *State) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			goal TEXT,
			tasks_json TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT 'active'
		);
		CREATE TABLE IF NOT EXISTS journal (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT,
			task_id TEXT,
			tool TEXT,
			args_json TEXT,
			output TEXT,
			error TEXT,
			status TEXT,
			cost REAL DEFAULT 0,
			tokens INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);
		CREATE INDEX IF NOT EXISTS idx_journal_session_task
			ON journal(session_id, task_id);
		CREATE INDEX IF NOT EXISTS idx_journal_session_status
			ON journal(session_id, status);
	`)
	if err != nil {
		return fmt.Errorf("state migration: %w", err)
	}
	return nil
}

func (s *State) Close() error { return s.db.Close() }

// CreateSession creates a new session and returns its ID.
func (s *State) CreateSession(goal string) (string, error) {
	id := fmt.Sprintf("sess_%d", time.Now().UnixNano())
	_, err := s.db.Exec("INSERT INTO sessions (id, goal) VALUES (?, ?)", id, goal)
	return id, err
}

// SaveSessionTasks persists the task plan JSON so resume doesn't need to re-plan.
func (s *State) SaveSessionTasks(sessionID string, tasksJSON []byte) error {
	_, err := s.db.Exec("UPDATE sessions SET tasks_json = ? WHERE id = ?", string(tasksJSON), sessionID)
	return err
}

// GetSessionTasks retrieves the persisted task plan JSON (nil if not saved).
func (s *State) GetSessionTasks(sessionID string) ([]byte, error) {
	var raw sql.NullString
	err := s.db.QueryRow("SELECT tasks_json FROM sessions WHERE id = ?", sessionID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	return []byte(raw.String), nil
}

func (s *State) GetGoal(sessionID string) (string, error) {
	var goal string
	err := s.db.QueryRow("SELECT goal FROM sessions WHERE id = ?", sessionID).Scan(&goal)
	return goal, err
}

func (s *State) GetSessions() ([]string, error) {
	rows, err := s.db.Query("SELECT id FROM sessions ORDER BY created_at DESC LIMIT 20")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var sessions []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan session id: %w", err)
		}
		sessions = append(sessions, id)
	}
	return sessions, rows.Err()
}

func (s *State) GetFailedTasks(sessionID string) ([]string, error) {
	rows, err := s.db.Query(
		"SELECT DISTINCT task_id FROM journal WHERE session_id = ? AND status = 'failed'",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var tasks []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan task id: %w", err)
		}
		tasks = append(tasks, id)
	}
	return tasks, rows.Err()
}

func (s *State) IsTaskCompleted(sessionID, taskID string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM journal WHERE session_id = ? AND task_id = ? AND status = 'completed'",
		sessionID, taskID,
	).Scan(&count)
	return count > 0, err
}

// Log records a task execution journal entry. Args are stored as JSON.
func (s *State) Log(entry JournalEntry) error {
	argsJSON, err := json.Marshal(entry.Args)
	if err != nil {
		return fmt.Errorf("marshal args: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO journal
			(session_id, task_id, tool, args_json, output, error, status, cost, tokens)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.SessionID, entry.TaskID, entry.Tool, string(argsJSON),
		entry.Output, entry.Error, entry.Status, entry.Cost, entry.Tokens,
	)
	return err
}
