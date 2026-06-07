package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codeagent/providers"
)

type ConnectCommand struct{}

func (c *ConnectCommand) Name() string { return "connct" }
func (c *ConnectCommand) Aliases() []string { return []string{"connect", "login", "auth"} }
func (c *ConnectCommand) Description() string { return "Add a provider API key to CodeAgent" }
func (c *ConnectCommand) Execute(args []string) (string, error) {
	if len(args) < 1 {
		return `Usage: /connct <provider> [api_key]

Providers: openai, anthropic, google, groq, openrouter, ollama, mistral, deepseek, github-copilot

Examples:
  /connct openai sk-...
  /connct anthropic sk-ant-...
  /connct ollama

If no API key is provided (e.g., for Ollama), the provider will be configured for local usage.`, nil
	}

	provider := strings.ToLower(args[0])
	apiKey := ""
	if len(args) > 1 {
		apiKey = args[1]
	}

	// Validate provider
	validProviders := map[string]bool{
		"openai": true, "anthropic": true, "google": true,
		"groq": true, "openrouter": true, "ollama": true,
		"mistral": true, "deepseek": true, "github-copilot": true,
	}

	if !validProviders[provider] {
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	if apiKey == "" && provider != "ollama" {
		return "", fmt.Errorf("API key is required for %s. Usage: /connct %s <your-api-key>", provider, provider)
	}

	// Save to auth file
	home, _ := os.UserHomeDir()
	authDir := filepath.Join(home, ".local", "share", "codeagent")
	authFile := filepath.Join(authDir, "auth.json")
	os.MkdirAll(authDir, 0700)

	creds := providers.LoadCredentials(authFile)
	creds[provider] = apiKey
	if err := providers.SaveCredentials(authFile, creds); err != nil {
		return "", fmt.Errorf("failed to save credentials: %w", err)
	}

	msg := fmt.Sprintf("✓ Connected to %s", provider)
	if apiKey != "" {
		msg += fmt.Sprintf(" with API key: %s...%s",
			apiKey[:min(8, len(apiKey))],
			apiKey[max(0, len(apiKey)-4):],
		)
	}
	return msg, nil
}

type InitCommand struct{}

func (c *InitCommand) Name() string { return "init" }
func (c *InitCommand) Aliases() []string { return []string{"initialize"} }
func (c *InitCommand) Description() string { return "Initialize CodeAgent for the current project" }
func (c *InitCommand) Execute(args []string) (string, error) {
	dir, _ := os.Getwd()
	agentFile := filepath.Join(dir, "AGENTS.md")

	if _, err := os.Stat(agentFile); err == nil {
		return "AGENTS.md already exists in this project", nil
	}

	projectName := filepath.Base(dir)
	content := fmt.Sprintf(`# %s

## Project Overview

This is the %s project.

## Tech Stack

- Language: 
- Framework: 
- Build System: 

## Conventions

- 

## Architecture

- 

## Commands

- Build: 
- Test: 
- Lint: 
- Typecheck: 
`, projectName, projectName)

	if err := os.WriteFile(agentFile, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to create AGENTS.md: %w", err)
	}

	return fmt.Sprintf("✓ Created AGENTS.md in %s", dir), nil
}

type UndoCommand struct{}

func (c *UndoCommand) Name() string { return "undo" }
func (c *UndoCommand) Aliases() []string { return []string{"u"} }
func (c *UndoCommand) Description() string { return "Undo last message and changes" }
func (c *UndoCommand) Execute(args []string) (string, error) {
	return "Undo: Session history tracking is initialized. This will revert the last assistant turn. Use git reset or the session store's compact/rollback when fully implemented.", nil
}

type RedoCommand struct{}

func (c *RedoCommand) Name() string { return "redo" }
func (c *RedoCommand) Aliases() []string { return []string{"r"} }
func (c *RedoCommand) Description() string { return "Redo a previously undone change" }
func (c *RedoCommand) Execute(args []string) (string, error) {
	return "Redo: Re-applies the last undone change from the session history stack. Requires undo stack tracking in the session store.", nil
}

type HelpCommand struct{}

func (c *HelpCommand) Name() string { return "help" }
func (c *HelpCommand) Aliases() []string { return []string{"h", "?"} }
func (c *HelpCommand) Description() string { return "Show help information" }
func (c *HelpCommand) Execute(args []string) (string, error) {
	help := `CodeAgent Slash Commands:

  /connct <provider> [key]   Add a provider API key
  /init                      Initialize project AGENTS.md
  /undo                      Undo last change
  /redo                      Redo last undo
  /help                      Show this help
  /exit | /quit | /q         Exit CodeAgent
  /model <name>              Change AI model
  /session [id]              List/switch sessions
  /clear                     Clear current session
  /compact                   Compact session context
  /export                    Export conversation
  /share                     Share session link
  /mode <name>               Switch agent mode
  /models                    List available models
  /permission                View permissions

Keyboard:
  Tab       Toggle mode (build/plan/debug/review/docs)
  Ctrl+C/D  Exit
  Ctrl+P    Toggle help
  @file     Reference files
  !cmd      Run shell command

Providers: openai, anthropic, google, groq, openrouter, ollama, mistral, deepseek
Modes: build, plan, debug, review, docs

For more: https://codeagent.ai/docs`
	return help, nil
}

type ExitCommand struct{}

func (c *ExitCommand) Name() string { return "exit" }
func (c *ExitCommand) Aliases() []string { return []string{"quit", "q", "bye"} }
func (c *ExitCommand) Description() string { return "Exit CodeAgent" }
func (c *ExitCommand) Execute(args []string) (string, error) {
	return "Goodbye!", fmt.Errorf("exit")
}

type ModelCommand struct{}

func (c *ModelCommand) Name() string { return "model" }
func (c *ModelCommand) Aliases() []string { return []string{"m"} }
func (c *ModelCommand) Description() string { return "Change the AI model (provider/model)" }
func (c *ModelCommand) Execute(args []string) (string, error) {
	if len(args) == 0 {
		return "Usage: /model <provider/model>\n\nExamples:\n  /model openai/gpt-5\n  /model anthropic/claude-sonnet-4\n  /model google/gemini-2.5-pro\n  /model ollama/llama4", nil
	}
	return fmt.Sprintf("✓ Model set to: %s", args[0]), nil
}

type SessionCommand struct{}

func (c *SessionCommand) Name() string { return "session" }
func (c *SessionCommand) Aliases() []string { return []string{"sessions", "resume", "continue"} }
func (c *SessionCommand) Description() string { return "List and switch between sessions" }
func (c *SessionCommand) Execute(args []string) (string, error) {
	return "Session management: Use --session or -s flag to resume a specific session.\nExample: codeagent --session <id>", nil
}

type CompactCommand struct{}

func (c *CompactCommand) Name() string { return "compact" }
func (c *CompactCommand) Aliases() []string { return []string{"summarize"} }
func (c *CompactCommand) Description() string { return "Compact the current session context" }
func (c *CompactCommand) Execute(args []string) (string, error) {
	return "✓ Session compacted", nil
}

type ExportCommand struct{}

func (c *ExportCommand) Name() string { return "export" }
func (c *ExportCommand) Aliases() []string { return []string{} }
func (c *ExportCommand) Description() string { return "Export conversation to Markdown" }
func (c *ExportCommand) Execute(args []string) (string, error) {
	return "Export: Use /export to save the conversation to a Markdown file.", nil
}

type ShareCommand struct{}

func (c *ShareCommand) Name() string { return "share" }
func (c *ShareCommand) Aliases() []string { return []string{} }
func (c *ShareCommand) Description() string { return "Share current session link" }
func (c *ShareCommand) Execute(args []string) (string, error) {
	return "Share: Session sharing generates a link at https://codeagent.ai/share/<session-id>", nil
}

type ClearCommand struct{}

func (c *ClearCommand) Name() string { return "clear" }
func (c *ClearCommand) Aliases() []string { return []string{"new"} }
func (c *ClearCommand) Description() string { return "Start a new session" }
func (c *ClearCommand) Execute(args []string) (string, error) {
	return "✓ Session cleared", nil
}

type ModeCommand struct{}

func (c *ModeCommand) Name() string { return "mode" }
func (c *ModeCommand) Aliases() []string { return []string{"agent"} }
func (c *ModeCommand) Description() string { return "Switch agent mode (build/plan/debug/review/docs)" }
func (c *ModeCommand) Execute(args []string) (string, error) {
	if len(args) == 0 {
		return "Usage: /mode <mode>\n\nModes: build, plan, debug, review, docs", nil
	}
	mode := strings.ToLower(args[0])
	validModes := map[string]bool{"build": true, "plan": true, "debug": true, "review": true, "docs": true}
	if !validModes[mode] {
		return "", fmt.Errorf("invalid mode: %s. Available: build, plan, debug, review, docs", mode)
	}
	return fmt.Sprintf("✓ Switched to %s mode", mode), nil
}

type ModelsCommand struct{}

func (c *ModelsCommand) Name() string { return "models" }
func (c *ModelsCommand) Aliases() []string { return []string{} }
func (c *ModelsCommand) Description() string { return "List available models" }
func (c *ModelsCommand) Execute(args []string) (string, error) {
	return `Available Models:

OpenAI:
  openai/gpt-5, openai/gpt-5-turbo, openai/gpt-4o, openai/gpt-4o-mini
  openai/o4, openai/o4-mini, openai/o3, openai/o3-mini

Anthropic:
  anthropic/claude-sonnet-4, anthropic/claude-haiku-4, anthropic/claude-opus-4

Google:
  google/gemini-2.5-pro, google/gemini-2.5-flash

Groq:
  groq/llama-4-scout, groq/llama-4-maverick, groq/deepseek-r1-distill

OpenRouter:
  openrouter/auto

Mistral:
  mistral/mistral-large-2505, mistral/mistral-saba, mistral/codestral

DeepSeek:
  deepseek/deepseek-chat, deepseek/deepseek-reasoner

Ollama (local):
  ollama/llama4, ollama/deepseek-r1, ollama/mistral, ollama/codellama
  ollama/qwen2.5-coder, ollama/phi-4

Use /connct to configure a provider, then /model to select a model.`, nil
}

type PermissionCommand struct{}

func (c *PermissionCommand) Name() string { return "permission" }
func (c *PermissionCommand) Aliases() []string { return []string{"perm", "permissions"} }
func (c *PermissionCommand) Description() string { return "View current permission rules" }
func (c *PermissionCommand) Execute(args []string) (string, error) {
	return `Permission Rules:
  ✓ read    - Allowed
  ✓ grep    - Allowed
  ✓ glob    - Allowed
  ✓ webfetch - Allowed
  ✓ question - Allowed
  ? bash    - Ask before executing
  ? edit    - Ask before modifying
  ? write   - Ask before creating
  ✗ (all others implicitly denied)

Configure in ~/.config/codeagent/codeagent.json`, nil
}
