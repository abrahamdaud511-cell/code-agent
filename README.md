[README.md](https://github.com/user-attachments/files/28675722/README.md)
# CodeAgent

**The open source AI coding agent**

CodeAgent is an open source AI coding agent that helps you write code in your terminal. It supports 75+ LLM providers including OpenAI, Anthropic, Google, Ollama, and more.

[![GitHub stars](https://img.shields.io/github/stars/anomalyco/codeagent)](https://github.com/anomalyco/codeagent)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/anomalyco/codeagent)](go.mod)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)


### npm / bun
```bash
npm install -g codeagent-ai
# or
bun install -g codeagent-ai
```

### Scoop (Windows)
```powershell
scoop install codeagent
```

### Chocolatey (Windows)
```powershell
choco install codeagent
```

## Quick Start

```bash
# Start CodeAgent in your project
codeagent

# Or run with a prompt
codeagent run "Explain this codebase"

# Configure a provider
codeagent auth login --provider openai --key sk-...
```

## Usage

### Terminal User Interface (TUI)

Run `codeagent` in your project directory to start the interactive TUI:

```
  CodeAgent [BUILD]
  ┌─────────────────────────────────────┐
  │ You: What does this code do?        │
  │                                     │
  │ CodeAgent: Let me analyze the       │
  │ codebase...                         │
  │                                     │
  │ The main function initializes the   │
  │ server and handles requests...      │
  │                                     │
  └─────────────────────────────────────┘
  > Ask CodeAgent to do anything...
   BUILD mode | /help for commands
```

### Modes

Press `Tab` to switch between modes:

| Mode | Description |
|------|-------------|
| **Build** | Full access - read, write, execute |
| **Plan** | Read-only - analyze and plan |
| **Debug** | Investigation - find bugs |
| **Review** | Code review - quality check |
| **Docs** | Documentation - write and maintain |

### Slash Commands

| Command | Description |
|---------|-------------|
| `/connct <provider> <key>` | Add provider API key |
| `/init` | Initialize project AGENTS.md |
| `/model <name>` | Change model |
| `/mode <name>` | Switch agent mode |
| `/undo` | Undo last change |
| `/redo` | Redo last undo |
| `/clear` | Clear session |
| `/help` | Show help |
| `/exit` | Exit CodeAgent |
| `/session [id]` | List/switch sessions |
| `/compact` | Compact context |
| `/export` | Export conversation |
| `/share` | Share session |
| `/models` | List available models |
| `/permission` | View permissions |

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Tab` | Toggle mode |
| `Ctrl+C/D` | Exit |
| `Ctrl+P` | Toggle help |
| `Ctrl+N` | New session |
| `Up/Down` | Scroll |
| `PgUp/PgDown` | Page scroll |
| `@file` | Reference a file |
| `!command` | Run shell command |

## Supported Providers

CodeAgent supports 19 providers through a provider-agnostic architecture:

| Provider | Models | Authentication |
|----------|--------|----------------|
| **OpenAI** | GPT-5, GPT-4o, o4, o3 | `OPENAI_API_KEY` |
| **Anthropic** | Claude Sonnet 4, Opus 4, Haiku 4 | `ANTHROPIC_API_KEY` |
| **Google** | Gemini 2.5 Pro, 2.5 Flash | `GOOGLE_API_KEY` |
| **Groq** | Llama 4, DeepSeek R1 | `GROQ_API_KEY` |
| **OpenRouter** | 200+ models | `OPENROUTER_API_KEY` |
| **Mistral** | Mistral Large, Codestral | `MISTRAL_API_KEY` |
| **DeepSeek** | DeepSeek Chat, Reasoner | `DEEPSEEK_API_KEY` |
| **Ollama** | Local models (Llama 4, Qwen, etc.) | No key needed |
| **GitHub Copilot** | Copilot models | GitHub login |
| **AWS Bedrock** | Claude, Llama | AWS credentials |
| **Azure** | GPT-4o, GPT-5 | `AZURE_API_KEY` |
| **Cohere** | Command R+, Command A | `COHERE_API_KEY` |
| **Perplexity** | Sonar Pro, Sonar | `PERPLEXITY_API_KEY` |
| **xAI** | Grok 3, Grok 3 Mini | `XAI_API_KEY` |
| **TogetherAI** | Llama 4, DeepSeek V3 | `TOGETHERAI_API_KEY` |
| **DeepInfra** | Llama 4, DeepSeek V3 | `DEEPINFRA_API_KEY` |
| **Cerebras** | Llama 4 Scout | `CEREBRAS_API_KEY` |
| **Alibaba (Qwen)** | Qwen Max, Plus, Turbo | `ALIBABA_API_KEY` |
| **Venice** | Llama 4, DeepSeek R1 (free) | `VENICE_API_KEY` |

### Using /connct in TUI

Inside the TUI, use the `/connct` command to add your API keys:

```
/connct openai sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
/connct anthropic sk-ant-xxxxxxxxxxxxxxxxxxxx
/connct ollama
```

Or set environment variables:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
codeagent
```

## Configuration

CodeAgent uses a layered configuration system:

1. System config (`/etc/codeagent/codeagent.json`)
2. User config (`~/.config/codeagent/codeagent.json`)
3. Project config (`./codeagent.json` or `./.codeagent/codeagent.json`)
4. Environment variables

Example `~/.config/codeagent/codeagent.json`:

```json
{
  "default_model": "openai/gpt-5",
  "default_provider": "openai",
  "theme": "catppuccin-mocha",
  "log_level": "INFO",
  "permissions": [
    { "tool": "bash", "action": "ask" },
    { "tool": "read", "action": "allow" },
    { "tool": "edit", "action": "ask" }
  ]
}
```

## Project Initialization

Run `/init` in the TUI or `codeagent init` to create an `AGENTS.md` file that helps CodeAgent understand your project:

```markdown
# My Project

## Tech Stack
- Language: TypeScript
- Framework: Next.js
- Build System: Turborepo

## Commands
- Build: npm run build
- Test: npm test
- Lint: npm run lint
```

## CLI Reference

```bash
codeagent                     # Start TUI
codeagent tui                 # Start TUI (explicit)
codeagent run <prompt>        # Non-interactive mode
codeagent serve               # Start HTTP server
codeagent web                 # Start web interface
codeagent auth login          # Add provider
codeagent auth list           # List providers
codeagent auth remove <prov>  # Remove provider
codeagent init                # Init project
codeagent models              # List models
codeagent agent create        # Create custom agent
codeagent mcp add             # Add MCP server
codeagent version             # Show version
codeagent --help              # Show help
```

### Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Config file path |
| `--log-level` | Log level (DEBUG, INFO, WARN, ERROR) |
| `--print-logs` | Print logs to stderr |
| `--pure` | Run without plugins |
| `-h, --help` | Show help |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `CODEAGENT_CONFIG` | Config file path |
| `CODEAGENT_LOG_LEVEL` | Log level |
| `CODEAGENT_DEFAULT_MODEL` | Default model |
| `CODEAGENT_THEME` | UI theme |
| `CODEAGENT_DATA_DIR` | Data directory |
| `CODEAGENT_SERVER_PASSWORD` | Server auth password |
| `OPENAI_API_KEY` | OpenAI API key |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `GOOGLE_API_KEY` | Google API key |
| `GROQ_API_KEY` | Groq API key |
| `OPENROUTER_API_KEY` | OpenRouter API key |

## Architecture

CodeAgent uses a client-server architecture:

```
┌─────────────────────────────────────────┐
│              Client (TUI)                │
│  ┌─────────┐  ┌────────┐  ┌─────────┐  │
│  │  Chat   │  │ Input  │  │ Status  │  │
│  │  View   │  │  Bar   │  │  Bar    │  │
│  └────┬────┘  └────────┘  └─────────┘  │
└───────┼─────────────────────────────────┘
        │ RPC / SSE
┌───────┼─────────────────────────────────┐
│  ┌────┴────────────────────────────┐    │
│  │         Agent Loop              │    │
│  │  ┌────────┐ ┌────────┐ ┌─────┐ │    │
│  │  │  LLM   │ │ Tools  │ │Perms│ │    │
│  │  └────────┘ └────────┘ └─────┘ │    │
│  └─────────────────────────────────┘    │
│  ┌────────────┐  ┌───────────────────┐  │
│  │   SQLite   │  │  HTTP Server      │  │
│  │  Sessions  │  │  (REST + SSE)     │  │
│  └────────────┘  └───────────────────┘  │
└─────────────────────────────────────────┘
```

### Key Components

- **Agent Loop**: Core loop that manages LLM calls, tool execution, and response handling
- **Provider System**: Unified interface for 75+ LLM providers
- **Tool System**: 11 built-in tools (bash, read, write, edit, grep, glob, webfetch, websearch, question, task, todowrite)
- **Permission System**: Layered rules (allow, deny, ask) for tool access control
- **Plugin System**: Go plugin support for extending functionality
- **Skill System**: YAML-based skill definitions for agent behavior
- **Event Bus**: Pub/sub event system connecting all components
- **Credential Store**: AES-256-GCM encrypted credential storage
- **Session Management**: SQLite-backed session persistence with multi-session support
- **LSP Integration**: Automatic LSP detection and diagnostics
- **MCP Support**: Model Context Protocol for external tools

## Development

```bash
# Prerequisites
go install github.com/mattn/go-sqlite3

# Clone
git clone https://github.com/anomalyco/codeagent.git
cd codeagent

# Build
go build -o codeagent .

# Run
./codeagent

# Test
go test ./...

# Cross-compile (with CGO for SQLite)
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o codeagent-linux-amd64 .
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o codeagent-darwin-amd64 .
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -o codeagent-windows-amd64.exe .
```

## Project Structure

```
codeagent/
├── server.go              # Server entry point
├── main.go                # CLI entry point
├── api/                   # HTTP server, routes, SSE
├── cli/                   # CLI command definitions
├── config/                # Configuration system
├── core/                  # Core logic (sub-packages)
│   ├── agent/             # Agent loop and orchestration
│   ├── bus/               # Event bus system
│   ├── command/           # Slash command handlers
│   ├── crypto/            # Credential encryption
│   ├── lsp/               # Language Server Protocol
│   ├── permission/        # Tool permission system
│   ├── plugin/            # Plugin manager
│   ├── session/           # Session management + SQLite
│   ├── skill/             # Skill manager
│   ├── tool/              # 11 built-in tools
│   └── tui/               # Terminal User Interface
├── messaging/             # Discord/Telegram adapters
├── providers/             # 19 LLM providers
├── scripts/               # Install/build scripts
├── smoke/                 # Smoke tests
└── .github/               # CI/CD workflows
```

## Publishing

```bash
# Build for all platforms
./scripts/build.sh

# Create GitHub release
gh release create v1.0.0 ./dist/* --title "v1.0.0" --notes "Release notes"
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Links

- [Website](https://codeagent.ai)
- [Documentation](https://codeagent.ai/docs)
- [GitHub](https://github.com/anomalyco/codeagent)
- [Discord](https://discord.gg/codeagent)

---

Built with ❤️ for developers who refuse to be locked into a single AI provider.
BY : ABRAHAM DAUD
##IG : 
**@abraham_daud01** 
##Gmail : 
**abrahamdaud511@gmail.com**
