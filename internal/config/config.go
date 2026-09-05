package config

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Marwanmorsy999/pivot/internal/paths"
)

type Config struct {
	Planner struct {
		Provider string `yaml:"provider"`
		Model    string `yaml:"model"`
		APIKey   string `yaml:"api_key"`
		Endpoint string `yaml:"endpoint"`
	} `yaml:"planner"`
	Worktree struct {
		Enabled bool   `yaml:"enabled"`
		BaseDir string `yaml:"base_dir"`
	} `yaml:"worktree"`
	Cost struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"cost"`
}

// DetectionResult holds auto-detected providers and local tools.
type DetectionResult struct {
	Providers        map[string]bool `yaml:"providers"`
	LocalTools       map[string]bool `yaml:"local_tools"`
	DetectedProvider string          `yaml:"detected_provider"`
	DetectedModel    string          `yaml:"detected_model"`
	DetectedEndpoint string          `yaml:"detected_endpoint"`
	DetectedAPIKey   string          `yaml:"detected_api_key"`
}

func Detect() *DetectionResult {
	r := &DetectionResult{
		Providers:  make(map[string]bool),
		LocalTools: make(map[string]bool),
	}
	r.detectProviders()
	r.detectLocalTools()
	r.pickBestProvider()
	return r
}

func (r *DetectionResult) detectProviders() {
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		r.Providers["openai"] = true
		r.DetectedAPIKey = k
	}
	if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
		r.Providers["anthropic"] = true
		if r.DetectedAPIKey == "" {
			r.DetectedAPIKey = k
		}
	}
	if k := os.Getenv("GROQ_API_KEY"); k != "" {
		r.Providers["groq"] = true
		if r.DetectedAPIKey == "" {
			r.DetectedAPIKey = k
		}
	}
	if k := os.Getenv("GEMINI_API_KEY"); k != "" || os.Getenv("GOOGLE_API_KEY") != "" {
		r.Providers["gemini"] = true
		if k == "" {
			k = os.Getenv("GOOGLE_API_KEY")
		}
		if r.DetectedAPIKey == "" && k != "" {
			r.DetectedAPIKey = k
		}
	}
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err == nil && resp != nil {
		_ = resp.Body.Close()
		r.Providers["ollama"] = true
	} else if _, err := exec.LookPath("ollama"); err == nil {
		r.Providers["ollama"] = true
	}
}

func (r *DetectionResult) detectLocalTools() {
	tools := []string{"git", "docker", "node", "python3", "python", "go", "kubectl", "npm", "curl", "jq", "az", "docker-compose"}
	for _, t := range tools {
		if _, err := exec.LookPath(t); err == nil {
			r.LocalTools[t] = true
		}
	}
}

func (r *DetectionResult) pickBestProvider() {
	if r.Providers["anthropic"] {
		r.DetectedProvider = "anthropic"
		r.DetectedModel = "claude-sonnet-4-5" // Sonnet: best cost/capability for automation
		r.DetectedEndpoint = "https://api.anthropic.com/v1/messages"
		return
	}
	if r.Providers["openai"] {
		r.DetectedProvider = "openai"
		r.DetectedModel = "gpt-4o-mini"
		r.DetectedEndpoint = "https://api.openai.com/v1/chat/completions"
		return
	}
	if r.Providers["groq"] {
		r.DetectedProvider = "groq"
		r.DetectedModel = "llama-3.1-8b-instant"
		r.DetectedEndpoint = "https://api.groq.com/openai/v1/chat/completions"
		return
	}
	if r.Providers["gemini"] {
		r.DetectedProvider = "gemini"
		r.DetectedModel = "gemini-1.5-flash"
		r.DetectedEndpoint = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
		return
	}
	if r.Providers["ollama"] {
		r.DetectedProvider = "ollama"
		r.DetectedModel = "llama3.2:3b"
		r.DetectedEndpoint = "http://localhost:11434"
	}
}

// Load reads config from disk then overlays API keys from environment variables.
// API keys are never written to disk; they must be set in the environment.
func Load() (*Config, error) {
	cfgFile, err := paths.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}
	data, err := os.ReadFile(cfgFile) // #nosec G304 -- PIVOT_HOME is an explicit user-configured data directory
	if err != nil {
		if os.IsNotExist(err) {
			cfg := defaultConfig()
			overlayEnvKeys(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	if cfg.Worktree.BaseDir == "" {
		cfg.Worktree.BaseDir = filepath.Join(os.TempDir(), "pivot-worktrees")
	}
	// API key on disk is always empty (stripped by saveConfig).
	// Always overlay from env so keys set after pivot init still work.
	overlayEnvKeys(&cfg)
	return &cfg, nil
}

// overlayEnvKeys fills in API keys from environment variables.
// Called after loading config from disk since keys are never persisted.
func overlayEnvKeys(cfg *Config) {
	switch cfg.Planner.Provider {
	case "anthropic":
		if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
			cfg.Planner.APIKey = k
		}
	case "openai":
		if k := os.Getenv("OPENAI_API_KEY"); k != "" {
			cfg.Planner.APIKey = k
		}
	case "groq":
		if k := os.Getenv("GROQ_API_KEY"); k != "" {
			cfg.Planner.APIKey = k
		}
	case "gemini":
		if k := os.Getenv("GEMINI_API_KEY"); k != "" {
			cfg.Planner.APIKey = k
		} else if k := os.Getenv("GOOGLE_API_KEY"); k != "" {
			cfg.Planner.APIKey = k
		}
	}
}

func defaultConfig() *Config {
	cfg := &Config{}
	cfg.Planner.Provider = "ollama"
	cfg.Planner.Model = "llama3.2:3b"
	cfg.Planner.Endpoint = "http://localhost:11434"
	cfg.Worktree.Enabled = false
	cfg.Worktree.BaseDir = filepath.Join(os.TempDir(), "pivot-worktrees")
	cfg.Cost.Enabled = true
	return cfg
}

func ConfigFromDetection(r *DetectionResult) *Config {
	cfg := defaultConfig()
	if r.DetectedProvider != "" {
		cfg.Planner.Provider = r.DetectedProvider
	}
	if r.DetectedModel != "" {
		cfg.Planner.Model = r.DetectedModel
	}
	if r.DetectedEndpoint != "" {
		cfg.Planner.Endpoint = r.DetectedEndpoint
	}
	if r.DetectedAPIKey != "" {
		cfg.Planner.APIKey = r.DetectedAPIKey
	}
	// Worktree mode requires a git repo in CWD — leave disabled by default.
	// Users opt in via config or --worktree flag.
	return cfg
}

func SaveDetected(cfg *Config) error {
	return saveConfig(cfg)
}

func SaveDefault() error {
	cfgFile, err := paths.ConfigFile()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		return saveConfig(defaultConfig())
	}
	return nil
}

func saveConfig(cfg *Config) error {
	cfgFile, err := paths.ConfigFile()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfgFile), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	// Never write API keys to disk — they must live in env vars.
	safe := *cfg
	safe.Planner.APIKey = ""
	data, err := yaml.Marshal(safe)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(cfgFile, data, 0600); err != nil { // #nosec G306
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
