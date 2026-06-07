# CodeAgent Architecture

## Overview

CodeAgent uses a layered, modular architecture designed for extensibility and cross-platform support.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Client Layer                            │
│  ┌────────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │  TUI (Bubbletea)│  │  Web UI      │  │  IDE Extensions │  │
│  └───────┬────────┘  └──────┬───────┘  └────────┬────────┘  │
└──────────┼───────────────────┼───────────────────┼──────────┘
           │                   │                   │
┌──────────┼───────────────────┼───────────────────┼──────────┐
│          │    HTTP/SSE API   │                   │          │
│  ┌───────┴───────────────────┴───────────────────┴──────┐  │
│  │                   Server Layer                        │  │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────────────┐   │  │
│  │  │ REST API │  │ SSE Bus  │  │ Session Manager   │   │  │
│  │  └────┬─────┘  └──────────┘  └────────┬──────────┘   │  │
│  └───────┼───────────────────────────────┼──────────────┘  │
└──────────┼───────────────────────────────┼─────────────────┘
           │                               │
┌──────────┼───────────────────────────────┼─────────────────┐
│          │         Agent Layer           │                  │
│  ┌───────┴───────────────────────────────┴──────────────┐  │
│  │                   Agent Loop                          │  │
│  │  ┌──────────┐  ┌──────────┐  ┌─────┐  ┌──────────┐  │  │
│  │  │ LLM      │  │ Tool     │  │Perm │  │ Message  │  │  │
│  │  │ Provider │  │ Registry │  │Mgr  │  │ Pipeline │  │  │
│  │  └──────────┘  └──────────┘  └─────┘  └──────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                 Provider Layer                        │  │
│  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────────┐  │  │
│  │  │OpenAI│ │Anthr │ │Google│ │Groq  │ │ Ollama   │  │  │
│  │  │      │ │opic  │ │      │ │      │ │ (local)  │  │  │
│  │  └──────┘ └──────┘ └──────┘ └──────┘ └──────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                Storage Layer                          │  │
│  │  ┌────────────────┐  ┌────────────────────────────┐  │  │
│  │  │  SQLite (per-  │  │  Config (Multi-Layer)     │  │  │
│  │  │  project)      │  │  JSONC + Env + CLI        │  │  │
│  │  └────────────────┘  └────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Agent Loop (`internal/agent/`)

The agent loop is the heart of CodeAgent. It:

1. Receives user input
2. Builds messages with system prompt and history
3. Sends to LLM provider
4. Processes responses (text or tool calls)
5. Executes tools and returns results
6. Repeats until completion or max iterations

```go
for i := 0; i < maxIterations; i++ {
    messages := buildMessages()
    resp := provider.Chat(ctx, &ChatRequest{
        Messages: messages,
        Tools:    toolDefinitions,
    })
    
    if len(resp.ToolCalls) == 0 {
        return resp.Content  // Done
    }
    
    for _, tc := range resp.ToolCalls {
        result := toolRegistry.Execute(tc.Name, tc.Args)
        session.AddToolResult(tc.ID, result)
    }
}
```

### 2. Provider System (`internal/llm/`)

All LLM providers implement a common interface:

```go
type Provider interface {
    Name() ProviderType
    Models() ([]ModelInfo, error)
    Chat(ctx, *ChatRequest) (*ChatResponse, error)
    ChatStream(ctx, *ChatRequest) (<-chan StreamEvent, error)
}
```

Each provider handles its own API format internally, converting to/from the standard `ChatRequest`/`ChatResponse` types.

### 3. Tool System (`internal/tool/`)

Tools are the agent's interface to the external world:

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() interface{}
    Execute(ctx, json.RawMessage) (string, error)
}
```

Tools are registered in a central `Registry` and exposed to the LLM as function definitions.

### 4. Configuration System (`internal/config/`)

Seven-layer configuration merge (lowest to highest priority):

1. `/etc/codeagent/codeagent.json` (system admin)
2. `~/.config/codeagent/codeagent.json` (user global)
3. `CODEAGENT_CONFIG` env var path
4. `./codeagent.json` (project root)
5. `./.codeagent/codeagent.json` (local overrides)
6. `CODEAGENT_CONFIG_CONTENT` (inline JSON)
7. Environment variables

### 5. Session Management (`internal/session/`)

- SQLite-backed persistence (WAL mode for concurrency)
- Per-project database isolation
- Session CRUD operations
- Context compaction
- Fork/undo/redo support
- Export to Markdown

### 6. Permission System (`internal/permission/`)

Three actions for each tool:
- `allow` - Always permitted
- `deny` - Always blocked
- `ask` - Requires user approval

Rules are defined in config and support glob pattern matching.

### 7. TUI Layer (`internal/tui/`)

Built with Bubbletea framework:
- Chat view with Markdown rendering (glamour)
- Input bar with slash command support
- Status bar showing current mode
- Multi-mode support (build/plan/debug/review/docs)
- File reference autocomplete (@)
- Shell command execution (!)

## Data Flow

### Chat Request Flow

```
User Input -> TUI Input -> Agent Loop
    -> Build Messages (system prompt + history)
    -> LLM Provider (via Provider interface)
    -> Parse Response
    -> If text: return to user
    -> If tool calls: execute tools -> add results -> loop
```

### Event Flow

```
Agent Loop -> Bus.Publish(event)
    -> Local subscribers (in-process)
    -> GlobalBus -> Worker Thread
        -> SSE -> TUI/Web clients
```

## Security Model

- API keys stored in `~/.local/share/codeagent/auth.json` (0600 permissions)
- Optional server-side password protection
- Tool-level permission controls
- Git-backed change tracking for undo/redo
- No code or context data stored on external servers

## Extensibility

### Adding Providers

Create a new file in `internal/llm/` implementing the `Provider` interface, then register in `NewProvider()`.

### Adding Tools

Create a new file in `internal/tool/` implementing the `Tool` interface, then register in `NewRegistry()`.

### MCP Integration

External tools via Model Context Protocol are automatically registered in the same tool registry as built-in tools, treated identically by the agent.
