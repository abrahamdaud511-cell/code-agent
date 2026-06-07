package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultModel != "openai/gpt-5" {
		t.Errorf("expected default model openai/gpt-5, got %s", cfg.DefaultModel)
	}
	if cfg.Theme != "catppuccin-mocha" {
		t.Errorf("expected default theme catppuccin-mocha, got %s", cfg.Theme)
	}
	if cfg.LogLevel != "INFO" {
		t.Errorf("expected default log level INFO, got %s", cfg.LogLevel)
	}
	if len(cfg.Permissions) == 0 {
		t.Error("expected default permissions to be non-empty")
	}
}

func TestConfigMerge(t *testing.T) {
	base := DefaultConfig()
	override := &Config{
		DefaultModel: "anthropic/claude-sonnet-4",
		Theme:        "dracula",
		LogLevel:     "DEBUG",
	}
	base.merge(override)

	if base.DefaultModel != "anthropic/claude-sonnet-4" {
		t.Errorf("expected merged model, got %s", base.DefaultModel)
	}
	if base.Theme != "dracula" {
		t.Errorf("expected merged theme, got %s", base.Theme)
	}
	if base.LogLevel != "DEBUG" {
		t.Errorf("expected merged log level, got %s", base.LogLevel)
	}
}

func TestConfigLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "codeagent.json")

	configData := `{
		"default_model": "test/model",
		"theme": "nord",
		"log_level": "DEBUG",
		"permissions": [
			{"tool": "bash", "action": "deny"}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	if err := cfg.loadFromFile(configPath); err != nil {
		t.Fatal(err)
	}

	if cfg.DefaultModel != "test/model" {
		t.Errorf("expected test/model, got %s", cfg.DefaultModel)
	}
	if cfg.Theme != "nord" {
		t.Errorf("expected nord theme, got %s", cfg.Theme)
	}
}

func TestConfigWithComments(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "codeagent.json")

	configData := `{
		// This is a comment
		"default_model": "openai/gpt-5",
		/* Block comment */
		"theme": "catppuccin-mocha"
	}`

	if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	if err := cfg.loadFromFile(configPath); err != nil {
		t.Fatal(err)
	}

	if cfg.DefaultModel != "openai/gpt-5" {
		t.Errorf("expected openai/gpt-5, got %s", cfg.DefaultModel)
	}
}

func TestProviderNames(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Providers["openai"] = ProviderConfig{APIKey: "test-key"}
	cfg.Providers["anthropic"] = ProviderConfig{APIKey: "test-key-2"}

	names := cfg.ProviderNames()
	if len(names) != 2 {
		t.Errorf("expected 2 provider names, got %d: %v", len(names), names)
	}
}

func TestIsPathAllowed(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.IsPathAllowed("/any/path") {
		t.Error("expected all paths allowed when no restrictions")
	}

	cfg.AllowedPaths = []string{"/home/user/project"}
	if !cfg.IsPathAllowed("/home/user/project/main.go") {
		t.Error("expected allowed path to be permitted")
	}
	if cfg.IsPathAllowed("/etc/passwd") {
		t.Error("expected non-allowed path to be denied")
	}

	cfg.DeniedPaths = []string{"/home/user/project/secret"}
	if !cfg.IsPathAllowed("/home/user/project/main.go") {
		t.Error("expected non-denied path to be allowed")
	}
}

func TestStripComments(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"key": "value"}`, `{"key": "value"}`},
		{`{"key": "value"} // comment`, `{"key": "value"} `},
		{`{"key": "value" /* comment */}`, `{"key": "value" }`},
		{`{"key": "value // not comment"}`, `{"key": "value // not comment"}`},
	}

	for _, tt := range tests {
		result := stripComments(tt.input)
		if result != tt.expected {
			t.Errorf("stripComments(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestConfigRoundtrip(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.ConfigDir = dir
	cfg.DefaultModel = "test/model"
	cfg.Theme = "dracula"

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	savedPath := filepath.Join(dir, "codeagent.json")
	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Fatal("saved config file does not exist")
	}

	data, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatal(err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.DefaultModel != "test/model" {
		t.Errorf("expected test/model, got %s", loaded.DefaultModel)
	}
}
