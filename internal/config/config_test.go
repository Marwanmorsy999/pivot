package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	if strings.Contains(string(data), "sk-") {
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
	r := &DetectionResult{
		LocalTools: map[string]bool{"git": true, "docker": true},
	}
	cfg := ConfigFromDetection(r)
	if cfg.Worktree.Enabled {
		t.Error("worktree should remain disabled by default even when git+docker present")
	}
}

// ── parseOllamaList ──────────────────────────────────────────────────────────

func TestParseOllamaList_Empty(t *testing.T) {
	models := parseOllamaList("")
	if len(models) != 0 {
		t.Errorf("expected no models from empty input, got %d", len(models))
	}
}

func TestParseOllamaList_HeaderOnly(t *testing.T) {
	models := parseOllamaList("NAME\tID\tSIZE\tMODIFIED\n")
	if len(models) != 0 {
		t.Errorf("expected no models (header only), got %d", len(models))
	}
}

func TestParseOllamaList_SingleModel(t *testing.T) {
	out := "NAME               ID         SIZE   MODIFIED\nllama3.2:3b        abc123     2.0 GB 2 days ago\n"
	models := parseOllamaList(out)
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].Name != "llama3.2:3b" {
		t.Errorf("expected name llama3.2:3b, got %q", models[0].Name)
	}
	if models[0].Provider != "ollama" {
		t.Errorf("expected provider ollama, got %q", models[0].Provider)
	}
	if models[0].Endpoint != "http://localhost:11434" {
		t.Errorf("unexpected endpoint %q", models[0].Endpoint)
	}
}

func TestParseOllamaList_MultipleModels(t *testing.T) {
	out := "NAME               ID         SIZE   MODIFIED\nllama3.2:3b        abc123     2.0 GB 2 days ago\nmistral:7b         def456     4.1 GB 1 week ago\ncodellama:13b      ghi789     7.4 GB 3 days ago\n"
	models := parseOllamaList(out)
	if len(models) != 3 {
		t.Fatalf("expected 3 models, got %d: %v", len(models), models)
	}
	names := make(map[string]bool)
	for _, m := range models {
		names[m.Name] = true
	}
	for _, want := range []string{"llama3.2:3b", "mistral:7b", "codellama:13b"} {
		if !names[want] {
			t.Errorf("expected model %q not found in %v", want, models)
		}
	}
}

func TestParseOllamaList_BlankLinesIgnored(t *testing.T) {
	out := "NAME               ID         SIZE   MODIFIED\nllama3.2:3b        abc123     2.0 GB 2 days ago\n\n   \nmistral:7b         def456     4.1 GB 1 week ago\n"
	models := parseOllamaList(out)
	if len(models) != 2 {
		t.Errorf("expected 2 models (blank lines ignored), got %d", len(models))
	}
}

// ── fetchOllamaModels ────────────────────────────────────────────────────────

func TestFetchOllamaModels_Success(t *testing.T) {
	payload := map[string]interface{}{
		"models": []map[string]interface{}{
			{"name": "llama3.2:3b", "size": int64(2_000_000_000)},
			{"name": "mistral:7b", "size": int64(4_100_000_000)},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	r := &DetectionResult{Providers: map[string]bool{}, LocalModels: nil}
	models := r.fetchOllamaModels(srv.URL)
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Provider != "ollama" {
		t.Errorf("expected provider ollama, got %q", models[0].Provider)
	}
	if models[0].Endpoint != srv.URL {
		t.Errorf("endpoint mismatch: got %q", models[0].Endpoint)
	}
	if models[0].SizeGB < 1.9 || models[0].SizeGB > 2.1 {
		t.Errorf("unexpected SizeGB %f", models[0].SizeGB)
	}
}

func TestFetchOllamaModels_ServerDown(t *testing.T) {
	r := &DetectionResult{Providers: map[string]bool{}}
	// Point at a port that nothing is listening on.
	models := r.fetchOllamaModels("http://127.0.0.1:19999")
	if models != nil {
		t.Errorf("expected nil from unreachable server, got %v", models)
	}
}

func TestFetchOllamaModels_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{not valid json"))
	}))
	defer srv.Close()

	r := &DetectionResult{Providers: map[string]bool{}}
	models := r.fetchOllamaModels(srv.URL)
	if models != nil {
		t.Errorf("expected nil for malformed JSON, got %v", models)
	}
}

// ── fetchOpenAIModels ────────────────────────────────────────────────────────

func TestFetchOpenAIModels_Success(t *testing.T) {
	payload := map[string]interface{}{
		"data": []map[string]string{
			{"id": "gpt-4o"},
			{"id": "gpt-4o-mini"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	r := &DetectionResult{Providers: map[string]bool{}}
	models := r.fetchOpenAIModels("lmstudio", srv.URL)
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Provider != "lmstudio" {
		t.Errorf("expected provider lmstudio, got %q", models[0].Provider)
	}
	names := make(map[string]bool)
	for _, m := range models {
		names[m.Name] = true
	}
	if !names["gpt-4o"] || !names["gpt-4o-mini"] {
		t.Errorf("unexpected model names: %v", models)
	}
}

func TestFetchOpenAIModels_ServerDown(t *testing.T) {
	r := &DetectionResult{Providers: map[string]bool{}}
	models := r.fetchOpenAIModels("lmstudio", "http://127.0.0.1:19999")
	if models != nil {
		t.Errorf("expected nil from unreachable server, got %v", models)
	}
}

func TestFetchOpenAIModels_EmptyData(t *testing.T) {
	payload := map[string]interface{}{"data": []map[string]string{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	r := &DetectionResult{Providers: map[string]bool{}}
	models := r.fetchOpenAIModels("jan", srv.URL)
	if len(models) != 0 {
		t.Errorf("expected 0 models for empty data, got %d", len(models))
	}
}
