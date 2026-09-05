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

func TestSaveConfig_DoesNotPersistAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PIVOT_HOME", tmpDir)

	cfg := defaultConfig()
	cfg.Planner.Provider = "anthropic"
	cfg.Planner.APIKey = "sk-should-not-appear"

	if err := SaveDetected(cfg); err != nil {
		t.Fatalf("SaveDetected: %v", err)
	}

	path := filepath.Join(tmpDir, ".pivot", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if len(data) > 0 && contains(string(data), "sk-") {
		t.Errorf("API key written to config file — security regression\nfile contents:\n%s", data)
	}
}

func TestLoad_OverlaysEnvAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PIVOT_HOME", tmpDir)
	t.Setenv("ANTHROPIC_API_KEY", "sk-from-env")

	// Write config with anthropic provider but no key
	path := filepath.Join(tmpDir, ".pivot", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("planner:\n  provider: anthropic\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Planner.APIKey != "sk-from-env" {
		t.Errorf("expected API key from ANTHROPIC_API_KEY env, got %q", cfg.Planner.APIKey)
	}
}

func TestLoad_CorruptFile_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PIVOT_HOME", tmpDir)

	path := filepath.Join(tmpDir, ".pivot", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("this: is: {not: [valid yaml"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Error("expected error for corrupt config file, got nil")
	}
}

func TestConfigFromDetection_WorktreeDisabled(t *testing.T) {
	r := DetectionResult{
		LocalTools: map[string]bool{"git": true, "docker": true},
	}
	cfg := ConfigFromDetection(r)
	if cfg.Worktree.Enabled {
		t.Error("worktree should remain disabled by default even when git+docker present")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsHelper(s, sub))
}

func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
