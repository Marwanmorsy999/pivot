package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Create() (string, error) {
	baseDir := filepath.Join(os.TempDir(), "pivot-worktrees")
	os.MkdirAll(baseDir, 0755)

	dir, err := os.MkdirTemp(baseDir, "wt-*")
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", "worktree", "add", dir, "-b", "pivot-temp")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("git worktree add failed: %w", err)
	}

	return dir, nil
}

func Cleanup(dir string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", dir)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree remove failed: %w", err)
	}
	return nil
}
