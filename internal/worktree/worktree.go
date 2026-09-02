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
	baseDir := filepath.Join(os.TempDir(), "pivot-worktrees")
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return "", fmt.Errorf("create worktree base dir: %w", err)
	}

	// MkdirTemp gives us a unique path like /tmp/pivot-worktrees/wt-123456789
	dir, err := os.MkdirTemp(baseDir, "wt-*")
	if err != nil {
		return "", fmt.Errorf("create worktree temp dir: %w", err)
	}

	// Derive a unique branch name from the temp dir basename so that two
	// concurrent runs don't both try to create "pivot-temp".
	branchName := "pivot-" + filepath.Base(dir)

	// #nosec G204 -- executable is "git" (hardcoded), dir comes from os.MkdirTemp
	cmd := exec.Command("git", "worktree", "add", dir, "-b", branchName)
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
	// #nosec G204 -- executable is "git", dir was returned by Create()
	if err := exec.Command("git", "worktree", "remove", "--force", dir).Run(); err != nil {
		return fmt.Errorf("git worktree remove: %w", err)
	}
	// Delete the unique branch we created.
	branchName := "pivot-" + filepath.Base(dir)
	_ = exec.Command("git", "branch", "-D", branchName).Run() // #nosec G204 -- branch name derived from MkdirTemp
	return nil
}
