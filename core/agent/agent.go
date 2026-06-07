package agent

import (
	"context"
	"fmt"
	"time"

	"codeagent/core/bus"
	"codeagent/config"
	"codeagent/providers"
	"codeagent/core/plugin"
	"codeagent/core/session"
	"codeagent/core/skill"
	"codeagent/core/tool"
)

type Agent struct {
	cfg          *config.Config
	session      *session.Session
	provider     providers.Provider
	tools        *tool.Registry
	pluginMgr    *plugin.Manager
	skillMgr     *skill.Manager
	ctx          context.Context
	cancel       context.CancelFunc
	interactive  bool
}

type Response struct {
	Text      string          `json:"text"`
	ToolCalls []providers.ToolCall  `json:"tool_calls,omitempty"`
	Usage     providers.Usage       `json:"usage,omitempty"`
}

func (r *Response) JSON() string {
	return fmt.Sprintf(`{"text":%q}`, r.Text)
}

func New(cfg *config.Config, sess *session.Session, provider providers.Provider) (*Agent, error) {
	tr := tool.NewRegistry(cfg.Permissions)
	tr.SetInteractive(false)

	ctx, cancel := context.WithCancel(context.Background())

	pluginMgr := plugin.NewManager(cfg.PluginDir)
	pluginMgr.LoadAll()

	skillMgr := skill.NewManager(cfg.SkillDir)
	skillMgr.LoadAll()

	return &Agent{
		cfg: cfg, session: sess, provider: provider,
		tools: tr, pluginMgr: pluginMgr, skillMgr: skillMgr,
		ctx: ctx, cancel: cancel,
	}, nil
}

func (a *Agent) SetInteractive(v bool) {
	a.interactive = v
	a.tools.SetInteractive(v)
}

func (a *Agent) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *Agent) Run(prompt string) (*Response, error) {
	a.session.AddMessage("user", prompt)

	bus.PublishEvent(bus.EventUserInput, map[string]string{
		"content": prompt,
		"session": a.session.ID,
	}, "agent")

	maxIter := 25
	for i := 0; i < maxIter; i++ {
		select {
		case <-a.ctx.Done():
			return nil, a.ctx.Err()
		default:
		}

		msgs := a.buildMessages()
		defs := a.tools.Definitions()

		req := &providers.ChatRequest{
			Model:    a.session.Model,
			Messages: msgs,
			Tools:    defs,
		}

		resp, err := a.provider.Chat(a.ctx, req)
		if err != nil {
			return nil, fmt.Errorf("LLM error: %w", err)
		}

		a.session.AddMessage("assistant", resp.Content)

		bus.PublishEvent(bus.EventAssistantResp, map[string]interface{}{
			"content": resp.Content,
			"usage":   resp.Usage,
		}, "agent")

		if len(resp.ToolCalls) == 0 {
			return &Response{
				Text:  resp.Content,
				Usage: resp.Usage,
			}, nil
		}

		for _, tc := range resp.ToolCalls {
			bus.PublishEvent(bus.EventToolCall, map[string]interface{}{
				"tool": tc.Function.Name,
				"args": tc.Function.Arguments,
			}, "agent")

			result, err := a.tools.Execute(a.ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			a.session.AddToolResult(tc.ID, result, err)

			bus.PublishEvent(bus.EventToolResult, map[string]interface{}{
				"tool":   tc.Function.Name,
				"result": result,
				"error":  err != nil,
			}, "agent")
		}
	}

	return nil, fmt.Errorf("reached max iterations (%d)", maxIter)
}

func (a *Agent) RunStream(prompt string) (<-chan providers.StreamEvent, error) {
	a.session.AddMessage("user", prompt)
	events := make(chan providers.StreamEvent, 100)

	go func() {
		defer close(events)

		msgs := a.buildMessages()
		defs := a.tools.Definitions()

		req := &providers.ChatRequest{
			Model:    a.session.Model,
			Messages: msgs,
			Tools:    defs,
			Stream:   true,
		}

		stream, err := a.provider.ChatStream(a.ctx, req)
		if err != nil {
			events <- providers.StreamEvent{Type: providers.StreamEventError, Error: err}
			return
		}

		var full string
		for event := range stream {
			switch event.Type {
			case providers.StreamEventText:
				full += event.Content
			case providers.StreamEventToolCall:
				if event.ToolCall != nil {
					r, err := a.tools.Execute(a.ctx, event.ToolCall.Function.Name, event.ToolCall.Function.Arguments)
					if err != nil {
						r = fmt.Sprintf("Error: %v", err)
					}
					a.session.AddToolResult(event.ToolCall.ID, r, err)
				}
			case providers.StreamEventDone:
				a.session.AddMessage("assistant", full)
			}
			events <- event
		}
	}()

	return events, nil
}

func (a *Agent) buildMessages() []providers.Message {
	msgs := []providers.Message{{Role: "system", Content: a.getSystemPrompt()}}

	for _, m := range a.session.GetHistory() {
		msg := providers.Message{Role: m.Role, Content: m.Content}

		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]providers.ToolCall, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				msg.ToolCalls[i] = providers.ToolCall{
					ID: tc.ID, Type: "function",
				}
				msg.ToolCalls[i].Function.Name = tc.Tool
				msg.ToolCalls[i].Function.Arguments = tc.Input
			}
		}

		if m.Role == "tool" && len(m.ToolResults) > 0 {
			msg.ToolCallID = m.ToolResults[0].ToolCallID
		}

		msgs = append(msgs, msg)
	}

	return msgs
}

func (a *Agent) getSystemPrompt() string {
	now := time.Now().Format("2006-01-02 15:04")

	// Add skill context
	skillContext := a.skillMgr.GetSystemPromptContext()

	switch a.session.AgentName {
	case "plan":
		return fmt.Sprintf(`You are CodeAgent in Plan mode — read-only analysis.
Date: %s

You CANNOT modify files. Only read, search, and analyze.
Provide step-by-step plans with file references.
Focus on:
- Architecture understanding
- Impact analysis
- Implementation strategy
- Risk assessment%s`, now, skillContext)

	case "debug":
		return fmt.Sprintf(`You are CodeAgent in Debug mode — issue investigation.
Date: %s

Focus on root cause analysis. Use bash (non-destructive) and read tools.
Approach:
1. Reproduce the issue
2. Isolate the root cause
3. Suggest a fix
4. Verify the fix%s`, now, skillContext)

	case "review":
		return fmt.Sprintf(`You are CodeAgent in Review mode — code review.
Date: %s

Analyze for:
- Bugs and logic errors
- Security vulnerabilities
- Performance issues
- Code style and conventions
- Test coverage
- Documentation completeness

You are read-only.%s`, now, skillContext)

	case "docs":
		return fmt.Sprintf(`You are CodeAgent in Docs mode — documentation.
Date: %s

You can read and write files. Focus on clear documentation.
Guidelines:
- Use Markdown format
- Include code examples
- Cover edge cases
- Keep it concise%s`, now, skillContext)

	default:
		return fmt.Sprintf(`You are CodeAgent, an open source AI coding agent.
Date: %s
Working directory: %s

You have these tools: bash, read, write, edit, grep, glob, webfetch, websearch, question, task, todowrite

Rules:
1. Read files before editing them
2. Ask before destructive commands (rm, force push)
3. Verify changes with tests when possible
4. Be concise and direct
5. Use AGENTS.md for project context
6. Reference files with full paths
7. For multi-step tasks, use the todowrite tool to track progress
8. For complex research tasks, use the task tool to delegate to sub-agents%s`, now, a.session.ProjectDir, skillContext)
	}
}
