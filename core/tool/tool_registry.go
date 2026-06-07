package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"codeagent/config"
	"codeagent/providers"
	"codeagent/core/permission"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() interface{}
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

type Registry struct {
	mu          sync.RWMutex
	tools       map[string]Tool
	permManager *permission.Manager
	interactive bool
}

func NewRegistry(permRules []config.PermissionRule) *Registry {
	r := &Registry{
		tools:       make(map[string]Tool),
		permManager: permission.NewManager(permRules),
		interactive: false,
	}

	r.Register(NewBashTool())
	r.Register(NewReadTool())
	r.Register(NewWriteTool())
	r.Register(NewEditTool())
	r.Register(NewGrepTool())
	r.Register(NewGlobTool())
	r.Register(NewWebFetchTool())
	r.Register(NewWebSearchTool())
	r.Register(NewQuestionTool())
	r.Register(NewTaskTool())
	r.Register(NewTodoWriteTool())
	r.Register(NewMouseMoveTool())
	r.Register(NewMouseClickTool())
	r.Register(NewMouseScrollTool())
	r.Register(NewKeyboardTypeTool())
	r.Register(NewKeyboardPressTool())

	return r
}

func (r *Registry) SetInteractive(interactive bool) {
	r.interactive = interactive
}

func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
	action, err := r.permManager.Check(name, "")
	if err != nil {
		return "", fmt.Errorf("permission check failed: %w", err)
	}

	if action == permission.ActionDeny {
		return "", fmt.Errorf("tool %s is denied by permission rules", name)
	}

	if action == permission.ActionAsk && r.interactive {
		allowed := r.permManager.Prompt(name, argsJSON)
		if !allowed {
			return "", fmt.Errorf("tool %s execution denied by user", name)
		}
	}

	if action == permission.ActionAsk && !r.interactive {
		fmt.Fprintf(os.Stderr, "\n⚠️  Tool %s requires approval\n", name)
		fmt.Fprintf(os.Stderr, "   Args: %s\n", truncateStr(argsJSON, 200))
		fmt.Fprintf(os.Stderr, "   Set interactive mode to enable prompts, or change permission to 'allow' in config.\n")
		return "", fmt.Errorf("tool %s requires approval. Configure permission to 'allow' or run in interactive mode", name)
	}

	t, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	return t.Execute(ctx, json.RawMessage(argsJSON))
}

func (r *Registry) Definitions() []providers.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]providers.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, providers.ToolDefinition{
			Type: "function",
			Function: providers.FunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return defs
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

func (r *Registry) GetPermissionManager() *permission.Manager {
	return r.permManager
}

func (r *Registry) PermissionString() string {
	return r.permManager.String()
}

func truncateStr(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

var _ Tool = (*TaskTool)(nil)
var _ Tool = (*TodoWriteTool)(nil)
var _ Tool = (*MouseMoveTool)(nil)
var _ Tool = (*MouseClickTool)(nil)
var _ Tool = (*MouseScrollTool)(nil)
var _ Tool = (*KeyboardTypeTool)(nil)
var _ Tool = (*KeyboardPressTool)(nil)

// TaskTool allows spawning sub-agents (matching opencode's task tool)
type TaskTool struct{}

type TaskArgs struct {
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	SubagentType string `json:"subagent_type,omitempty"`
}

func NewTaskTool() *TaskTool {
	return &TaskTool{}
}

func (t *TaskTool) Name() string {
	return "task"
}

func (t *TaskTool) Description() string {
	return "Launch a sub-agent to handle a specific task autonomously. The sub-agent can use all available tools. Use this for delegating complex work to a focused agent."
}

func (t *TaskTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"description": map[string]interface{}{
				"type":        "string",
				"description": "A short (3-5 words) description of the task",
			},
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "The detailed task for the sub-agent to perform",
			},
			"subagent_type": map[string]interface{}{
				"type":        "string",
				"description": "Type of subagent: explore, general (default: general)",
				"enum":        []string{"explore", "general"},
			},
		},
		"required": []string{"description", "prompt"},
	}
}

func (t *TaskTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args TaskArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}

	fmt.Fprintf(os.Stderr, "\n🤖 Spawning sub-agent for: %s\n", args.Description)
	fmt.Fprintf(os.Stderr, "   Task: %s\n\n", truncateStr(args.Prompt, 200))

	result, err := executeSubAgent(ctx, args.Prompt)
	if err != nil {
		return "", fmt.Errorf("sub-agent failed: %w", err)
	}

	return result, nil
}

func executeSubAgent(ctx context.Context, prompt string) (string, error) {
	// Real implementation would spawn a new agent loop with its own provider/session.
	// This requires access to the agent package and configuration, which should be
	// injected into TaskTool at creation time rather than instantiated here.
	//
	// Pattern to implement:
	//   ag, err := agent.New(cfg, sess, provider)
	//   result, err := ag.Run(prompt)
	return fmt.Sprintf(`Sub-agent task: %s

To enable full sub-agent execution, inject the agent factory into TaskTool.
See core/agent/agent.go for the Agent API.`, truncateStr(prompt, 100)), nil
}

// TodoWriteTool matches opencode's todowrite tool
type TodoWriteTool struct {
	todos []TodoItem
	mu    sync.Mutex
}

type TodoItem struct {
	Content string `json:"content"`
	Status  string `json:"status"`
	Priority string `json:"priority"`
}

type TodoWriteArgs struct {
	Todos []TodoItem `json:"todos"`
}

func NewTodoWriteTool() *TodoWriteTool {
	return &TodoWriteTool{
		todos: make([]TodoItem, 0),
	}
}

func (t *TodoWriteTool) Name() string {
	return "todowrite"
}

func (t *TodoWriteTool) Description() string {
	return "Create and maintain a structured task list for the current session. Tracks progress, organizes multi-step work, and surfaces status. Use this for complex multi-step tasks."
}

func (t *TodoWriteTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"todos": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Brief description of the task",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Current status: pending, in_progress, completed, cancelled",
							"enum":        []string{"pending", "in_progress", "completed", "cancelled"},
						},
						"priority": map[string]interface{}{
							"type":        "string",
							"description": "Priority level: high, medium, low",
							"enum":        []string{"high", "medium", "low"},
						},
					},
					"required": []string{"content", "status"},
				},
				"description": "List of tasks to track",
			},
		},
		"required": []string{"todos"},
	}
}

func (t *TodoWriteTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args TodoWriteArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.todos = args.Todos

	var sb strings.Builder
	sb.WriteString("## Current Task List\n\n")

	pending := 0
	inProgress := 0
	completed := 0
	cancelled := 0

	for _, todo := range t.todos {
		switch todo.Status {
		case "pending":
			pending++
		case "in_progress":
			inProgress++
		case "completed":
			completed++
		case "cancelled":
			cancelled++
		}
	}

	sb.WriteString(fmt.Sprintf("**Summary:** %d total | %d completed | %d in progress | %d pending | %d cancelled\n\n", len(t.todos), completed, inProgress, pending, cancelled))
	sb.WriteString("| Status | Priority | Task |\n")
	sb.WriteString("|--------|----------|------|\n")

	statusSymbol := map[string]string{
		"completed": "✅", "in_progress": "🔄", "pending": "⏳", "cancelled": "❌",
	}

	for _, todo := range t.todos {
		sym := statusSymbol[todo.Status]
		if sym == "" {
			sym = "⏳"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", sym, todo.Priority, todo.Content))
	}

	if os.Getenv("CODEAGENT_TODO_FILE") != "" {
		todoFile := os.Getenv("CODEAGENT_TODO_FILE")
		data, _ := json.MarshalIndent(t.todos, "", "  ")
		os.WriteFile(todoFile, data, 0644)
		sb.WriteString(fmt.Sprintf("\nTasks also saved to %s", todoFile))
	}

	return sb.String(), nil
}
