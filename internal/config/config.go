// Package config loads, validates, and writes safeops YAML configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "config.yaml"

// Config is the root safeops configuration.
type Config struct {
	Settings Settings            `yaml:"settings" json:"settings"`
	API      APIConfig           `yaml:"api" json:"api"`
	RAG      RAGConfig           `yaml:"rag" json:"rag"`
	Risk     map[string]RiskRule `yaml:"risk_rules" json:"risk_rules"`
}

// Settings controls safe defaults and policy thresholds.
type Settings struct {
	DefaultEngine            string  `yaml:"default_engine" json:"default_engine"`
	DefaultEnv               string  `yaml:"default_env" json:"default_env"`
	RequireProdConfirm       bool    `yaml:"require_prod_confirm" json:"require_prod_confirm"`
	MinGoalConfidenceToApply float64 `yaml:"min_goal_confidence_to_apply" json:"min_goal_confidence_to_apply"`
}

// APIConfig describes the optional reasoning provider.
type APIConfig struct {
	Provider string         `yaml:"provider,omitempty" json:"provider,omitempty"`
	DeepSeek DeepSeekConfig `yaml:"deepseek" json:"deepseek"`
}

// DeepSeekConfig contains DeepSeek-compatible client settings.
type DeepSeekConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	APIKey  string `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	BaseURL string `yaml:"base_url" json:"base_url"`
	Model   string `yaml:"model" json:"model"`
	Timeout int    `yaml:"timeout" json:"timeout"`
}

// RAGConfig controls local operational-document retrieval.
type RAGConfig struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`
	Paths        []string `yaml:"paths" json:"paths"`
	MaxDocuments int      `yaml:"max_documents" json:"max_documents"`
	MaxChars     int      `yaml:"max_chars" json:"max_chars"`
}

// RiskRule maps an Ansible module to safety guidance.
type RiskRule struct {
	Risk           string `yaml:"risk" json:"risk"`
	Reason         string `yaml:"reason" json:"reason"`
	Recommendation string `yaml:"recommendation" json:"recommendation"`
}

// Default returns a complete configuration with conservative behavior.
func Default() Config {
	return Config{
		Settings: Settings{
			DefaultEngine:            "ansible",
			DefaultEnv:               "dev",
			RequireProdConfirm:       true,
			MinGoalConfidenceToApply: 0.75,
		},
		API: APIConfig{DeepSeek: DeepSeekConfig{
			BaseURL: "https://api.deepseek.com",
			Model:   "deepseek-chat",
			Timeout: 30,
		}},
		RAG: RAGConfig{
			Paths:        []string{"docs/playbooks"},
			MaxDocuments: 3,
			MaxChars:     1200,
		},
		Risk: map[string]RiskRule{
			"shell":    {Risk: "HIGH", Reason: "runs arbitrary shell commands", Recommendation: "review the command and limit the host scope"},
			"service":  {Risk: "HIGH", Reason: "may change service state and availability", Recommendation: "use a staged rollout and verify service health"},
			"systemd":  {Risk: "HIGH", Reason: "may change systemd service state", Recommendation: "limit host scope and verify service impact"},
			"reboot":   {Risk: "HIGH", Reason: "reboots target machines", Recommendation: "use controlled batches and verify redundancy"},
			"copy":     {Risk: "MEDIUM", Reason: "modifies files on target hosts", Recommendation: "review dry-run diff before apply"},
			"template": {Risk: "MEDIUM", Reason: "renders and writes configuration files", Recommendation: "review generated diff and reload impact"},
			"file":     {Risk: "MEDIUM", Reason: "changes paths, ownership, or permissions", Recommendation: "verify paths and ownership before apply"},
			"debug":    {Risk: "LOW", Reason: "prints information only", Recommendation: "ensure output does not expose secrets"},
		},
	}
}

// Load reads path and overlays it on conservative defaults. A missing implicit
// config returns defaults; a missing explicit config is an error.
func Load(path string, explicit bool) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate rejects settings that could make policy behavior ambiguous.
func (c Config) Validate() error {
	if c.Settings.DefaultEngine == "" {
		return errors.New("settings.default_engine must not be empty")
	}
	threshold := c.Settings.MinGoalConfidenceToApply
	if threshold < 0 || threshold > 1 {
		return errors.New("settings.min_goal_confidence_to_apply must be between 0 and 1")
	}
	if c.API.DeepSeek.Timeout <= 0 {
		return errors.New("api.deepseek.timeout must be positive")
	}
	return nil
}

// WriteDefault creates a default config without overwriting an existing file.
func WriteDefault(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("configuration already exists: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect configuration path: %w", err)
	}
	data, err := yaml.Marshal(Default())
	if err != nil {
		return fmt.Errorf("encode configuration: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create configuration directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write configuration: %w", err)
	}
	return nil
}

// Masked returns a copy safe for display.
func (c Config) Masked() Config {
	masked := c
	if masked.API.DeepSeek.APIKey != "" {
		masked.API.DeepSeek.APIKey = "********"
	}
	return masked
}
