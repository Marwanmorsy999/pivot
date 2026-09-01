package state

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func pivotHome() (string, error) {
	if dir := os.Getenv("PIVOT_HOME"); dir != "" {
		return dir, nil
	}
	return os.UserHomeDir()
}

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

type State struct {
	db *sql.DB
}

func New() (*State, error) {
	home, err := pivotHome()
	if err != nil {
		return nil, fmt.Errorf("resolve pivot home: %w", err)
	}
	path := filepath.Join(home, ".pivot", "state.db") // #nosec G304 -- PIVOT_HOME is an explicit user-configured data directory.
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}
	db, err := sql.Open("sqlite3", path)
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
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT 'active'
		);
		CREATE TABLE IF NOT EXISTS journal (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT,
			task_id TEXT,
			tool TEXT,
			args TEXT,
			output TEXT,
			error TEXT,
			status TEXT,
			cost REAL DEFAULT 0,
			tokens INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		);
	`)
	if err != nil {
		return fmt.Errorf("state migration: %w", err)
	}
	return nil
}

func (s *State) Close() error {
	return s.db.Close()
}

func (s *State) CreateSession(goal string) (string, error) {
	id := fmt.Sprintf("sess_%d", time.Now().UnixNano())
	_, err := s.db.Exec("INSERT INTO sessions (id, goal) VALUES (?, ?)", id, goal)
	return id, err
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
	defer func() {
		_ = rows.Close()
	}()
	var sessions []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan session id: %w", err)
		}
		sessions = append(sessions, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return sessions, nil
}

func (s *State) GetFailedTasks(sessionID string) ([]string, error) {
	rows, err := s.db.Query("SELECT task_id FROM journal WHERE session_id = ? AND status = 'failed'", sessionID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	var tasks []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan task id: %w", err)
		}
		tasks = append(tasks, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate failed tasks: %w", err)
	}
	return tasks, nil
}

func (s *State) IsTaskCompleted(sessionID, taskID string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM journal WHERE session_id = ? AND task_id = ? AND status = 'completed'", sessionID, taskID).Scan(&count)
	return count > 0, err
}

func (s *State) Log(entry JournalEntry) error {
	argsStr := strings.Join(entry.Args, " ")
	_, err := s.db.Exec(
		"INSERT INTO journal (session_id, task_id, tool, args, output, error, status, cost, tokens) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		entry.SessionID, entry.TaskID, entry.Tool, argsStr, entry.Output, entry.Error, entry.Status, entry.Cost, entry.Tokens,
	)
	return err
}
