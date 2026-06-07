package session

import (
	"context"
	"fmt"

	"codeagent/config"
	"codeagent/providers"
	"codeagent/core/tool"
)

type AgentLoop struct {
	cfg      *config.Config
	session  *Session
	provider providers.Provider
	tools    *tool.Registry
}

func NewAgentLoop(cfg *config.Config, sess *Session, provider providers.Provider, tools *tool.Registry) *AgentLoop {
	return &AgentLoop{
		cfg:      cfg,
		session:  sess,
		provider: provider,
		tools:    tools,
	}
}

func (a *AgentLoop) Run(ctx context.Context, userInput string) (string, error) {
	a.session.AddMessage("user", userInput)

	maxIterations := 25
	for i := 0; i < maxIterations; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		messages := a.buildMessages()
		toolDefs := a.tools.Definitions()

		req := &providers.ChatRequest{
			Model:    a.session.Model,
			Messages: messages,
			Tools:    toolDefs,
		}

		resp, err := a.provider.Chat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("LLM chat failed: %w", err)
		}

		a.session.AddMessage("assistant", resp.Content)

		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		for _, tc := range resp.ToolCalls {
			result, err := a.tools.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}
			a.session.AddToolResult(tc.ID, result, err)
		}
	}

	return "", fmt.Errorf("max iterations reached")
}

func (a *AgentLoop) buildMessages() []providers.Message {
	msgs := make([]providers.Message, 0)

	msgs = append(msgs, providers.Message{
		Role: "system",
		Content: a.getSystemPrompt(),
	})

	for _, m := range a.session.GetHistory() {
		msg := providers.Message{
			Role:    m.Role,
			Content: m.Content,
		}

		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]providers.ToolCall, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				msg.ToolCalls[i] = providers.ToolCall{
					ID:   tc.ID,
					Type: "function",
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

func (a *AgentLoop) getSystemPrompt() string {
	agent := a.session.AgentName
	switch agent {
	case "plan":
		return `You are CodeAgent in Plan mode. You analyze code and create plans.
You have read-only access - you CANNOT modify files.
Focus on understanding the codebase and suggesting implementation strategies.

Available tools: read, grep, glob, webfetch, question`
	case "debug":
		return `You are CodeAgent in Debug mode. You investigate issues and find bugs.
You have read-only access for investigation.
Focus on root cause analysis and suggesting fixes.`
	case "review":
		return `You are CodeAgent in Review mode. You review code changes.
You have read-only access.
Focus on code quality, security, and best practices.`
	default:
		return `You are CodeAgent, an open source AI coding agent that helps users write code.
You have access to tools that can read, write, and execute commands in the project.

Rules:
1. Always ask before running destructive commands
2. Read files before editing them
3. Verify your changes with tests when possible
4. Use the AGENTS.md file for project context
5. Be concise and direct in your responses

Current project: ` + a.session.ProjectDir
	}
}
