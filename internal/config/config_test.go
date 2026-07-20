package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingImplicitConfigReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"), false)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Settings.DefaultEngine != "ansible" {
		t.Fatalf("DefaultEngine = %q, want ansible", cfg.Settings.DefaultEngine)
	}
	if !cfg.Settings.RequireProdConfirm {
		t.Fatal("RequireProdConfirm = false, want true")
	}
}

func TestLoadRejectsMissingExplicitConfig(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"), true)
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestWriteDefaultAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	if err := WriteDefault(path); err != nil {
		t.Fatalf("WriteDefault() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	if _, err := Load(path, true); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestMaskedHidesAPIKey(t *testing.T) {
	cfg := Default()
	cfg.API.DeepSeek.APIKey = "sk-secret"
	masked := cfg.Masked()
	if masked.API.DeepSeek.APIKey != "********" {
		t.Fatalf("masked key = %q", masked.API.DeepSeek.APIKey)
	}
	if cfg.API.DeepSeek.APIKey != "sk-secret" {
		t.Fatal("Masked() mutated the source config")
	}
}

func TestValidateConfidenceRange(t *testing.T) {
	cfg := Default()
	cfg.Settings.MinGoalConfidenceToApply = 1.1
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want range error")
	}
}
