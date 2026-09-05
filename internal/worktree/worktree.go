// Package worktree manages temporary git worktrees for isolated agent execution.
package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Create makes a temporary git worktree with a unique branch name so that
// concurrent pivot sessions don't collide on the same branch.
func Create() (string, error) {
	// Respect PIVOT_WORKTREE_DIR env or fall back to a temp subdir.
	baseDir := os.Getenv("PIVOT_WORKTREE_DIR")
	if baseDir == "" {
		baseDir = filepath.Join(os.TempDir(), "pivot-worktrees")
	}
	if err := os.MkdirAll(baseDir, 0700); err != nil { // #nosec G703 -- baseDir is user-controlled via PIVOT_WORKTREE_DIR env, intentional
		return "", fmt.Errorf("create worktree base dir: %w", err)
	}

	dir, err := os.MkdirTemp(baseDir, "wt-*") // #nosec G703 -- same justification
	if err != nil {
		return "", fmt.Errorf("create worktree temp dir: %w", err)
	}

	branchName := "pivot-" + filepath.Base(dir)

	cmd := exec.Command("git", "worktree", "add", dir, "-b", branchName) // #nosec G204,G702 -- "git" is hardcoded; dir/branchName from os.MkdirTemp (PIVOT_WORKTREE_DIR is user-trusted env)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if removeErr := os.RemoveAll(dir); removeErr != nil {
			return "", fmt.Errorf("git worktree add: %w (cleanup: %v)", err, removeErr)
		}
		return "", fmt.Errorf("git worktree add: %w", err)
	}

	return dir, nil
}

// Cleanup removes the worktree directory and deletes the temporary branch.
func Cleanup(dir string) error {
	if err := exec.Command("git", "worktree", "remove", "--force", dir).Run(); err != nil { // #nosec G204 -- "git" is hardcoded; dir was returned by Create()
		return fmt.Errorf("git worktree remove: %w", err)
	}
	branchName := "pivot-" + filepath.Base(dir)
	_ = exec.Command("git", "branch", "-D", branchName).Run() // #nosec G204 -- "git" hardcoded; branchName derived from MkdirTemp
	return nil
}
