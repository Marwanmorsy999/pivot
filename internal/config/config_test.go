package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Planner.Provider != "ollama" {
		t.Errorf("expected ollama, got %q", cfg.Planner.Provider)
	}
	if cfg.Planner.Model != "llama3.2:3b" {
		t.Errorf("expected llama3.2:3b, got %q", cfg.Planner.Model)
	}
	if cfg.Worktree.Enabled {
		t.Error("expected worktree disabled by default")
	}
	if !cfg.Cost.Enabled {
		t.Error("expected cost tracking enabled by default")
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PIVOT_HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Planner.Provider != "ollama" {
		t.Errorf("expected default provider, got %q", cfg.Planner.Provider)
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PIVOT_HOME", tmpDir)

	pivotDir := filepath.Join(tmpDir, ".pivot")
	if err := os.MkdirAll(pivotDir, 0755); err != nil {
		t.Fatalf("failed to create pivot dir: %v", err)
	}

	configData := `planner:
  provider: openai
  model: gpt-4o
  api_key: test-key
  endpoint: https://api.openai.com
worktree:
  enabled: true
  base_dir: /tmp/test
cost:
  enabled: false
`
	if err := os.WriteFile(filepath.Join(pivotDir, "config.yaml"), []byte(configData), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Planner.Provider != "openai" {
		t.Errorf("expected openai, got %q", cfg.Planner.Provider)
	}
	if cfg.Planner.APIKey != "test-key" {
		t.Errorf("expected test-key, got %q", cfg.Planner.APIKey)
	}
	if !cfg.Worktree.Enabled {
		t.Error("expected worktree enabled")
	}
	if cfg.Cost.Enabled {
		t.Error("expected cost disabled")
	}
}

func TestSaveDefault(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PIVOT_HOME", tmpDir)

	err := SaveDefault()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(tmpDir, ".pivot", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if len(data) == 0 {
		t.Error("config file is empty")
	}

	err = SaveDefault()
	if err != nil {
		t.Fatalf("second save should not error: %v", err)
	}
}
