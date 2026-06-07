package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type GitHubCopilotProvider struct {
	config      ProviderConfig
	model       string
	token       string
	tokenExpiry time.Time
}

func NewGitHubCopilotProvider(cfg ProviderConfig, model string) (*GitHubCopilotProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.githubcopilot.com"
	}
	return &GitHubCopilotProvider{
		config: ProviderConfig{
			APIKey:  cfg.APIKey,
			BaseURL: strings.TrimRight(baseURL, "/"),
		},
		model: model,
	}, nil
}

func (p *GitHubCopilotProvider) Name() ProviderType {
	return ProviderGitHubCopilot
}

func (p *GitHubCopilotProvider) Models() ([]ModelInfo, error) {
	return []ModelInfo{
		{Provider: "github-copilot", Name: "gpt-4o-copilot", ContextSize: 128000},
		{Provider: "github-copilot", Name: "claude-sonnet-4-copilot", ContextSize: 200000},
	}, nil
}

func (p *GitHubCopilotProvider) ensureToken() error {
	if p.token != "" && time.Now().Before(p.tokenExpiry) {
		return nil
	}

	token := p.config.APIKey
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		token = os.Getenv("GITHUB_COPILOT_TOKEN")
	}
	if token == "" {
		return fmt.Errorf("GitHub token required. Set GITHUB_TOKEN or GITHUB_COPILOT_TOKEN")
	}

	authReq := map[string]interface{}{
		"token": token,
	}

	body, _ := json.Marshal(authReq)
	resp, err := doJSONRequest("POST", "https://api.github.com/copilot_internal/v2/token",
		bytes.NewReader(body),
		map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
		})
	if err != nil {
		return err
	}

	var result struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := parseJSONResponse(resp, &result); err != nil {
		return fmt.Errorf("GitHub Copilot auth failed: %w", err)
	}

	p.token = result.Token
	p.tokenExpiry = time.Unix(result.ExpiresAt, 0)
	return nil
}

type copilotChatRequest struct {
	Model    string           `json:"model"`
	Messages []copilotMsg     `json:"messages"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
	Stream   bool             `json:"stream"`
}

type copilotMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type copilotChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Index        int        `json:"index"`
		Message      copilotMsg `json:"message"`
		FinishReason string     `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (p *GitHubCopilotProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if err := p.ensureToken(); err != nil {
		return nil, err
	}

	model := p.model
	if req.Model != "" {
		model = req.Model
	}

	messages := make([]copilotMsg, 0)
	for _, m := range req.Messages {
		messages = append(messages, copilotMsg{Role: m.Role, Content: m.Content})
	}

	chatReq := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}

	if len(req.Tools) > 0 {
		chatReq["tools"] = req.Tools
	}

	body, _ := json.Marshal(chatReq)

	headers := map[string]string{
		"Authorization":         "Bearer " + p.token,
		"Content-Type":          "application/json",
		"Editor-Version":        "CodeAgent/1.0",
		"Editor-Plugin-Version": "CodeAgent-1.0.0",
		"User-Agent":            "github.com/anomalyco/codeagent",
	}

	resp, err := doJSONRequest("POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(body), headers)
	if err != nil {
		return nil, err
	}

	var copilotResp copilotChatResponse
	if err := parseJSONResponse(resp, &copilotResp); err != nil {
		return nil, err
	}

	return &ChatResponse{
		Content:      copilotResp.Choices[0].Message.Content,
		FinishReason: copilotResp.Choices[0].FinishReason,
		Usage: Usage{
			PromptTokens:     copilotResp.Usage.PromptTokens,
			CompletionTokens: copilotResp.Usage.CompletionTokens,
			TotalTokens:      copilotResp.Usage.TotalTokens,
		},
	}, nil
}

func (p *GitHubCopilotProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	if err := p.ensureToken(); err != nil {
		return nil, err
	}

	model := p.model
	if req.Model != "" {
		model = req.Model
	}

	messages := make([]copilotMsg, 0)
	for _, m := range req.Messages {
		messages = append(messages, copilotMsg{Role: m.Role, Content: m.Content})
	}

	chatReq := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   true,
	}

	if len(req.Tools) > 0 {
		chatReq["tools"] = req.Tools
	}

	body, _ := json.Marshal(chatReq)

	headers := map[string]string{
		"Authorization":         "Bearer " + p.token,
		"Content-Type":          "application/json",
		"Editor-Version":        "CodeAgent/1.0",
		"Editor-Plugin-Version": "CodeAgent-1.0.0",
		"User-Agent":            "github.com/anomalyco/codeagent",
	}

	httpResp, err := doJSONRequest("POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(body), headers)
	if err != nil {
		return nil, err
	}

	events := make(chan StreamEvent, 100)
	go func() {
		defer httpResp.Body.Close()
		defer close(events)

		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				events <- StreamEvent{Type: StreamEventDone, Done: true}
				return
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					events <- StreamEvent{
						Type:    StreamEventText,
						Content: choice.Delta.Content,
					}
				}
				if choice.FinishReason != "" {
					events <- StreamEvent{Type: StreamEventDone, Done: true}
					return
				}
			}
		}
	}()

	return events, nil
}
