# Contributing to CodeAgent

We love your input! We want to make contributing to CodeAgent as easy and transparent as possible.

## Development Process

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code lints
6. Issue that pull request

## Go Code Style

- Run `gofmt` or `go fmt` before committing
- Follow standard Go conventions
- Use meaningful variable names
- Add comments for exported functions
- Wrap errors with context using `fmt.Errorf("...: %w", err)`

## Project Structure

- `cmd/` - CLI commands (Cobra)
- `internal/` - Internal packages
  - `agent/` - Agent loop
  - `command/` - Slash commands
  - `config/` - Configuration
  - `llm/` - Provider implementations
  - `permission/` - Permission system
  - `server/` - HTTP server
  - `session/` - Session management
  - `tool/` - Tool implementations
  - `tui/` - Terminal UI

## Adding a New Provider

1. Create a new file in `internal/llm/` (e.g., `cohere.go`)
2. Implement the `Provider` interface
3. Add the provider to `NewProvider()` in `provider.go`
4. Add default models in `defaultModels()` in `registry.go`
5. Add to `/connct` validation in `command/commands.go`

## Adding a New Tool

1. Create a new file in `internal/tool/` (e.g., `mytool.go`)
2. Implement the `Tool` interface
3. Register the tool in `NewRegistry()` in `registry.go`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
