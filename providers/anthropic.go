package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
)

type AnthropicProvider struct {
	config    ProviderConfig
	model     string
}

func NewAnthropicProvider(cfg ProviderConfig, model string) (*AnthropicProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	return &AnthropicProvider{
		config: ProviderConfig{
			APIKey:  cfg.APIKey,
			BaseURL: strings.TrimRight(baseURL, "/"),
		},
		model: model,
	}, nil
}

func (p *AnthropicProvider) Name() ProviderType {
	return ProviderAnthropic
}

func (p *AnthropicProvider) Models() ([]ModelInfo, error) {
	return []ModelInfo{
		{Provider: "anthropic", Name: "claude-sonnet-4"},
		{Provider: "anthropic", Name: "claude-haiku-4"},
		{Provider: "anthropic", Name: "claude-opus-4"},
	}, nil
}

type anthropicRequest struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	Messages    []anthropicMsg   `json:"messages"`
	System      string           `json:"system,omitempty"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	Stream      bool             `json:"stream"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

func (p *AnthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	antReq := p.toAnthropicRequest(req)
	body, err := json.Marshal(antReq)
	if err != nil {
		return nil, err
	}

	resp, err := doJSONRequest("POST", p.config.BaseURL+"/messages",
		bytes.NewReader(body),
		map[string]string{
			"x-api-key":         p.config.APIKey,
			"anthropic-version": "2023-06-01",
			"Content-Type":      "application/json",
		})
	if err != nil {
		return nil, err
	}

	var antResp anthropicResponse
	if err := parseJSONResponse(resp, &antResp); err != nil {
		return nil, err
	}

	return p.toChatResponse(&antResp), nil
}

func (p *AnthropicProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	antReq := p.toAnthropicRequest(req)
	antReq.Stream = true
	body, err := json.Marshal(antReq)
	if err != nil {
		return nil, err
	}

	httpResp, err := doJSONRequest("POST", p.config.BaseURL+"/messages",
		bytes.NewReader(body),
		map[string]string{
			"x-api-key":         p.config.APIKey,
			"anthropic-version": "2023-06-01",
			"Content-Type":      "application/json",
		})
	if err != nil {
		return nil, err
	}

	events := make(chan StreamEvent, 100)
	go p.processStream(ctx, httpResp.Body, events)
	return events, nil
}

func (p *AnthropicProvider) processStream(ctx context.Context, body io.ReadCloser, events chan<- StreamEvent) {
	defer body.Close()
	defer close(events)

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Text string `json:"text"`
			} `json:"delta"`
			ContentBlock *anthropicContentBlock `json:"content_block,omitempty"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta.Text != "" {
				events <- StreamEvent{
					Type:    StreamEventText,
					Content: event.Delta.Text,
				}
			}
		case "message_stop":
			events <- StreamEvent{Type: StreamEventDone, Done: true}
			return
		}
	}
}

func (p *AnthropicProvider) toAnthropicRequest(req *ChatRequest) *anthropicRequest {
	model := p.model
	if req.Model != "" {
		model = req.Model
	}

	messages := make([]anthropicMsg, 0)
	var system string

	for _, m := range req.Messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		messages = append(messages, anthropicMsg{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}

	return &anthropicRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		Messages:    messages,
		System:      system,
		Tools:       req.Tools,
		Temperature: req.Temperature,
	}
}

func (p *AnthropicProvider) toChatResponse(resp *anthropicResponse) *ChatResponse {
	cr := &ChatResponse{
		FinishReason: resp.StopReason,
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			cr.Content += block.Text
		case "tool_use":
			args, _ := json.Marshal(block.Input)
			tc := ToolCall{
				ID:   block.ID,
				Type: "function",
			}
			tc.Function.Name = block.Name
			tc.Function.Arguments = string(args)
			cr.ToolCalls = append(cr.ToolCalls, tc)
		}
	}

	return cr
}
