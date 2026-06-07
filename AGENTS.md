# CodeAgent

## Project Overview
CodeAgent is an open source AI coding agent that runs in the terminal. It helps developers write, debug, and refactor code using AI models from any provider.

## Tech Stack
- Language: Go
- TUI Framework: Bubbletea (charmbracelet/bubbletea)
- CLI Framework: Cobra (spf13/cobra)
- Database: SQLite (mattn/go-sqlite3)
- Config: Viper (spf13/viper)
- LLM SDK: Custom provider abstraction layer

## Directory Structure
```
codeagent/
├── server.go              # Server entry point (like free-claude-code server.py)
├── main.go                # CLI entry point
├── go.mod / go.sum
├── api/                   # HTTP server, routes, SSE (was internal/server/)
├── cli/                   # CLI command definitions (was cmd/)
├── config/                # Configuration system (was internal/config/)
├── core/                  # Core logic (sub-packages)
│   ├── agent/             # Agent loop and orchestration
│   ├── bus/               # Event bus system
│   ├── command/           # Slash command handlers
│   ├── crypto/            # Credential encryption
│   ├── lsp/               # Language Server Protocol integration
│   ├── permission/        # Tool permission system
│   ├── plugin/            # Plugin manager
│   ├── session/           # Session management and storage
│   ├── skill/             # Skill manager
│   ├── tool/              # Built-in tools (bash, read, write, edit, etc.)
│   └── tui/               # Terminal User Interface (Bubbletea)
├── messaging/             # Discord/Telegram adapters
├── providers/             # LLM provider abstraction layer (was internal/llm/)
├── scripts/               # Install/build scripts
├── smoke/                 # Smoke tests
├── assets/                # Images, diagrams
├── tests/                 # Additional tests
└── .github/               # CI/CD workflows
```

### Provider Support
Providers are abstracted through a common interface. Supported: openai, anthropic, google, groq, ollama, openrouter, mistral, deepseek, bedrock, azure, copilot, cohere, perplexity, xai, togetherai, deepinfra, cerebras, alibaba, venice

### Tool System
Tools are registered in a central registry and exposed to the LLM. Built-in tools: bash, read, write, edit, grep, glob, webfetch, websearch, question, task, todowrite

### Configuration Layers
1. `/etc/codeagent/codeagent.json` (admin)
2. `~/.config/codeagent/codeagent.json` (user)
3. `./codeagent.json` (project)
4. `./.codeagent.json` (project local)
5. Environment variables
6. CLI flags

## Build Commands
- Build: `go build -o codeagent .`
- Test: `go test ./...`
- Lint: `golangci-lint run`
- Cross-compile: `GOOS=linux GOARCH=amd64 go build -o codeagent-linux-amd64 .`

## Conventions
- Follow Go standard project layout
- Use interface-based abstractions for testability
- Errors should be wrapped with context
- JSONC config files (JSON with comments support)
- All provider implementations must satisfy the Provider interface
- Tools must implement the Tool interface and register in NewRegistry()
- Sessions use SQLite with WAL mode for concurrent access
