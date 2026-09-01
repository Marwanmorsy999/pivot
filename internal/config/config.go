package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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

func pivotHome() (string, error) {
	if dir := os.Getenv("PIVOT_HOME"); dir != "" {
		return dir, nil
	}
	return os.UserHomeDir()
}

func Load() (*Config, error) {
	home, _ := pivotHome()
	path := filepath.Join(home, ".pivot", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig(), nil
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	if cfg.Worktree.BaseDir == "" {
		cfg.Worktree.BaseDir = filepath.Join(os.TempDir(), "pivot-worktrees")
	}
	return &cfg, err
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

func SaveDefault() error {
	home, _ := pivotHome()
	dir := filepath.Join(home, ".pivot")
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "config.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := defaultConfig()
		data, _ := yaml.Marshal(cfg)
		return os.WriteFile(path, data, 0644)
	}
	return nil
}
