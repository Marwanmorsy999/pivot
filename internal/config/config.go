package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Marwanmorsy999/pivot/internal/paths"
)

// Config holds pivot's full configuration.
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

// LocalModel describes a model found on the device.
type LocalModel struct {
	Name     string // display name / model ID
	Provider string // ollama | lmstudio | llamafile | koboldcpp | jan | gguf
	Endpoint string // API base URL
	SizeGB   float64
	Path     string // for GGUF files
}

// DetectionResult holds auto-detected providers, local models, and tools.
type DetectionResult struct {
	Providers        map[string]bool `yaml:"providers"`
	LocalTools       map[string]bool `yaml:"local_tools"`
	LocalModels      []LocalModel    `yaml:"-"`
	DetectedProvider string          `yaml:"detected_provider"`
	DetectedModel    string          `yaml:"detected_model"`
	DetectedEndpoint string          `yaml:"detected_endpoint"`
	DetectedAPIKey   string          `yaml:"detected_api_key"`
}

// Detect runs full auto-detection: cloud providers, local servers, GGUF files, tools.
func Detect() *DetectionResult {
	r := &DetectionResult{
		Providers:  make(map[string]bool),
		LocalTools: make(map[string]bool),
	}
	r.detectCloudProviders()
	r.detectLocalServers()
	r.detectGGUFFiles()
	r.detectLocalTools()
	r.pickBestProvider()
	return r
}

// detectCloudProviders checks env vars for cloud API keys.
func (r *DetectionResult) detectCloudProviders() {
	if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
		r.Providers["anthropic"] = true
		if r.DetectedAPIKey == "" {
			r.DetectedAPIKey = k
		}
	}
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		r.Providers["openai"] = true
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
	if k := os.Getenv("GEMINI_API_KEY"); k != "" {
		r.Providers["gemini"] = true
		if r.DetectedAPIKey == "" {
			r.DetectedAPIKey = k
		}
	} else if k := os.Getenv("GOOGLE_API_KEY"); k != "" {
		r.Providers["gemini"] = true
		if r.DetectedAPIKey == "" {
			r.DetectedAPIKey = k
		}
	}
	if k := os.Getenv("MISTRAL_API_KEY"); k != "" {
		r.Providers["mistral"] = true
		if r.DetectedAPIKey == "" {
			r.DetectedAPIKey = k
		}
	}
	if k := os.Getenv("TOGETHER_API_KEY"); k != "" {
		r.Providers["together"] = true
		if r.DetectedAPIKey == "" {
			r.DetectedAPIKey = k
		}
	}
	if k := os.Getenv("OPENROUTER_API_KEY"); k != "" {
		r.Providers["openrouter"] = true
		if r.DetectedAPIKey == "" {
			r.DetectedAPIKey = k
		}
	}
}

// localServerProbe describes a local inference server to probe.
type localServerProbe struct {
	name     string
	port     int
	tagsPath string // GET this path to list models; empty = just check if reachable
	provider string // how pivot should talk to it
}

var localServers = []localServerProbe{
	{"ollama", 11434, "/api/tags", "ollama"},
	{"lmstudio", 1234, "/v1/models", "openai"},
	{"lmstudio-alt", 11434, "/v1/models", "openai"}, // LMStudio sometimes uses 11434
	{"jan", 1337, "/v1/models", "openai"},
	{"koboldcpp", 5001, "/api/v1/model", "openai"},
	{"llamafile", 8080, "/v1/models", "openai"},
	{"llamafile-alt", 8081, "/v1/models", "openai"},
	{"textgen-webui", 5000, "/v1/models", "openai"},
	{"localai", 8080, "/v1/models", "openai"},
}

// detectLocalServers probes known local inference server ports.
func (r *DetectionResult) detectLocalServers() {
	client := &http.Client{Timeout: 400 * time.Millisecond}
	seen := make(map[int]bool)

	for _, srv := range localServers {
		if seen[srv.port] {
			continue
		}
		url := fmt.Sprintf("http://localhost:%d%s", srv.port, srv.tagsPath) // #nosec G107
		resp, err := client.Get(url)
		if err != nil || resp == nil {
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 500 {
			continue
		}
		seen[srv.port] = true
		endpoint := fmt.Sprintf("http://localhost:%d", srv.port)

		// Mark provider available.
		if srv.name == "ollama" || strings.HasPrefix(srv.name, "ollama") {
			r.Providers["ollama"] = true
			// Fetch available Ollama models.
			r.LocalModels = append(r.LocalModels, r.fetchOllamaModels(endpoint)...)
		} else {
			// OpenAI-compatible server — add as generic local provider.
			r.Providers["local-openai"] = true
			r.LocalModels = append(r.LocalModels, r.fetchOpenAIModels(srv.name, endpoint)...)
		}
	}

	// Also check if ollama binary exists even if server isn't running.
	if !r.Providers["ollama"] {
		if _, err := exec.LookPath("ollama"); err == nil {
			r.Providers["ollama"] = true
			// Run ollama list to find available models.
			if out, err := exec.Command("ollama", "list").Output(); err == nil {
				for _, m := range parseOllamaList(string(out)) {
					r.LocalModels = append(r.LocalModels, m)
				}
			}
		}
	}
}

// fetchOllamaModels queries /api/tags and returns available models.
func (r *DetectionResult) fetchOllamaModels(endpoint string) []LocalModel {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(endpoint + "/api/tags") // #nosec G107
	if err != nil || resp == nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Models []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
		} `json:"models"`
	}
	if json.Unmarshal(body, &result) != nil {
		return nil
	}
	var models []LocalModel
	for _, m := range result.Models {
		models = append(models, LocalModel{
			Name:     m.Name,
			Provider: "ollama",
			Endpoint: endpoint,
			SizeGB:   float64(m.Size) / 1e9,
		})
	}
	return models
}

// fetchOpenAIModels queries /v1/models from an OpenAI-compatible server.
func (r *DetectionResult) fetchOpenAIModels(serverName, endpoint string) []LocalModel {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(endpoint + "/v1/models") // #nosec G107
	if err != nil || resp == nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &result) != nil {
		return nil
	}
	var models []LocalModel
	for _, m := range result.Data {
		models = append(models, LocalModel{
			Name:     m.ID,
			Provider: serverName,
			Endpoint: endpoint,
		})
	}
	return models
}

// parseOllamaList parses `ollama list` output into LocalModel entries.
func parseOllamaList(out string) []LocalModel {
	var models []LocalModel
	for i, line := range strings.Split(out, "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // skip header
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		models = append(models, LocalModel{
			Name:     fields[0],
			Provider: "ollama",
			Endpoint: "http://localhost:11434",
		})
	}
	return models
}

// ggufSearchDirs returns directories to scan for .gguf files.
func ggufSearchDirs() []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, ".ollama", "models"),
		filepath.Join(home, ".lmstudio", "models"),
		filepath.Join(home, "LMStudio", "models"),
		filepath.Join(home, ".jan", "models"),
		filepath.Join(home, "jan", "models"),
		filepath.Join(home, ".cache", "lm-studio", "models"),
		filepath.Join(home, "models"), // common convention
		filepath.Join(home, "gguf"),
		"/usr/share/ollama/models",
	}
	if runtime.GOOS == "darwin" {
		dirs = append(dirs,
			filepath.Join(home, "Library", "Application Support", "LM Studio", "models"),
			filepath.Join(home, "Library", "Application Support", "Jan", "models"),
		)
	}
	if runtime.GOOS == "windows" {
		appdata := os.Getenv("APPDATA")
		if appdata != "" {
			dirs = append(dirs,
				filepath.Join(appdata, "LM Studio", "models"),
				filepath.Join(appdata, "Jan", "models"),
			)
		}
	}
	return dirs
}

// detectGGUFFiles scans known model directories for .gguf files.
func (r *DetectionResult) detectGGUFFiles() {
	for _, dir := range ggufSearchDirs() {
		entries, err := os.ReadDir(dir) // #nosec G304
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".gguf") {
				continue
			}
			path := filepath.Join(dir, name)
			info, err := e.Info()
			if err != nil {
				continue
			}
			r.LocalModels = append(r.LocalModels, LocalModel{
				Name:     strings.TrimSuffix(name, ".gguf"),
				Provider: "gguf",
				Path:     path,
				SizeGB:   float64(info.Size()) / 1e9,
			})
		}
	}
}

// detectLocalTools checks for all tools in the allowed list plus extras.
func (r *DetectionResult) detectLocalTools() {
	tools := []string{
		// Unix core
		"find", "grep", "awk", "sed", "cat", "echo", "wc", "sort", "uniq",
		"head", "tail", "xargs", "tar", "zip", "unzip", "cut", "tr", "tee",
		"diff", "patch", "ls", "cp", "mv", "rm", "mkdir", "chmod", "touch",
		"stat", "env", "which", "date", "sleep", "curl", "wget", "jq", "ssh", "rsync",
		// Shell
		"bash", "sh", "zsh", "fish",
		// Dev
		"git", "make", "cmake",
		"go", "node", "python3", "python", "ruby", "java", "rustc", "cargo",
		"npm", "npx", "yarn", "pnpm", "pip", "pip3", "uv",
		// Containers & cloud
		"docker", "docker-compose", "podman", "kubectl", "helm", "terraform",
		"aws", "gcloud", "az",
		// AI agents
		"ollama", "claude-code", "gemini-cli", "mistral", "aider", "continue",
		// Misc useful
		"ffmpeg", "convert", "sqlite3", "psql", "mysql",
		"gh", // GitHub CLI
		"rg", // ripgrep
		"fd", // fd-find
		"bat", // batcat
		"delta", // git-delta
	}
	for _, t := range tools {
		if _, err := exec.LookPath(t); err == nil {
			r.LocalTools[t] = true
		}
	}
}

// pickBestProvider selects the best available provider, cloud first then local.
func (r *DetectionResult) pickBestProvider() {
	// Cloud providers in priority order.
	cloudPriority := []struct {
		provider, model, endpoint string
	}{
		{"anthropic", "claude-sonnet-4-5", "https://api.anthropic.com/v1/messages"},
		{"openai", "gpt-4o-mini", "https://api.openai.com/v1/chat/completions"},
		{"groq", "llama-3.1-8b-instant", "https://api.groq.com/openai/v1/chat/completions"},
		{"mistral", "mistral-small-latest", "https://api.mistral.ai/v1/chat/completions"},
		{"together", "meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo", "https://api.together.xyz/v1/chat/completions"},
		{"openrouter", "meta-llama/llama-3.1-8b-instruct:free", "https://openrouter.ai/api/v1/chat/completions"},
		{"gemini", "gemini-1.5-flash", "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"},
	}
	for _, c := range cloudPriority {
		if r.Providers[c.provider] {
			r.DetectedProvider = c.provider
			r.DetectedModel = c.model
			r.DetectedEndpoint = c.endpoint
			return
		}
	}

	// Local servers: pick best available model by size.
	for _, m := range r.LocalModels {
		if m.Provider == "ollama" {
			r.DetectedProvider = "ollama"
			r.DetectedModel = m.Name
			r.DetectedEndpoint = m.Endpoint
			return
		}
	}
	for _, m := range r.LocalModels {
		if m.Provider != "gguf" {
			r.DetectedProvider = "local-openai"
			r.DetectedModel = m.Name
			r.DetectedEndpoint = m.Endpoint
			return
		}
	}

	// Fallback: ollama binary with default model.
	if r.Providers["ollama"] {
		r.DetectedProvider = "ollama"
		r.DetectedModel = "llama3.2:3b"
		r.DetectedEndpoint = "http://localhost:11434"
	}
}

// BestLocalModel returns the best available local model name (largest non-GGUF, else any).
func (r *DetectionResult) BestLocalModel() *LocalModel {
	var best *LocalModel
	for i := range r.LocalModels {
		m := &r.LocalModels[i]
		if m.Provider == "gguf" {
			continue
		}
		if best == nil || m.SizeGB > best.SizeGB {
			best = m
		}
	}
	if best != nil {
		return best
	}
	for i := range r.LocalModels {
		return &r.LocalModels[i]
	}
	return nil
}

// Load reads config from disk then overlays API keys from environment variables.
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
	overlayEnvKeys(&cfg)
	return &cfg, nil
}

// overlayEnvKeys fills in API keys from environment variables.
func overlayEnvKeys(cfg *Config) {
	envMap := map[string][]string{
		"anthropic":   {"ANTHROPIC_API_KEY"},
		"openai":      {"OPENAI_API_KEY"},
		"groq":        {"GROQ_API_KEY"},
		"gemini":      {"GEMINI_API_KEY", "GOOGLE_API_KEY"},
		"mistral":     {"MISTRAL_API_KEY"},
		"together":    {"TOGETHER_API_KEY"},
		"openrouter":  {"OPENROUTER_API_KEY"},
		"local-openai": {"LM_STUDIO_API_KEY", "LOCALAI_API_KEY"},
	}
	for _, envVar := range envMap[cfg.Planner.Provider] {
		if k := os.Getenv(envVar); k != "" {
			cfg.Planner.APIKey = k
			return
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
	return cfg
}

func SaveDetected(cfg *Config) error { return saveConfig(cfg) }

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
