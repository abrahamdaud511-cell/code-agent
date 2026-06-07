package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type AzureProvider struct {
	config     ProviderConfig
	model      string
	deployment string
	endpoint   string
	apiVersion string
}

func NewAzureProvider(cfg ProviderConfig, model string) (*AzureProvider, error) {
	endpoint := cfg.BaseURL
	if endpoint == "" {
		endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("Azure endpoint required. Set AZURE_OPENAI_ENDPOINT or base_url")
	}

	apiVersion := os.Getenv("AZURE_OPENAI_API_VERSION")
	if apiVersion == "" {
		apiVersion = "2024-08-01-preview"
	}

	deployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	if deployment == "" {
		deployment = model
	}

	return &AzureProvider{
		config: ProviderConfig{
			APIKey:  cfg.APIKey,
			BaseURL: strings.TrimRight(endpoint, "/"),
		},
		model:      model,
		deployment: deployment,
		endpoint:   endpoint,
		apiVersion: apiVersion,
	}, nil
}

func (p *AzureProvider) Name() ProviderType {
	return ProviderAzure
}

func (p *AzureProvider) Models() ([]ModelInfo, error) {
	return []ModelInfo{
		{Provider: "azure", Name: "gpt-4o", ContextSize: 128000},
		{Provider: "azure", Name: "gpt-4o-mini", ContextSize: 128000},
		{Provider: "azure", Name: "gpt-5", ContextSize: 128000},
		{Provider: "azure", Name: "o4", ContextSize: 200000},
		{Provider: "azure", Name: "o4-mini", ContextSize: 200000},
	}, nil
}

func (p *AzureProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	apiKey := p.config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("AZURE_OPENAI_API_KEY")
	}

	messages := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		msg := openAIMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]openAIToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				msg.ToolCalls[j] = openAIToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		messages[i] = msg
	}

	body := map[string]interface{}{
		"messages": messages,
	}

	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}

	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		p.config.BaseURL, p.deployment, p.apiVersion)

	headers := map[string]string{
		"api-key":      apiKey,
		"Content-Type": "application/json",
	}

	resp, err := doJSONRequest("POST", url, bytes.NewReader(jsonBody), headers)
	if err != nil {
		return nil, err
	}

	var openAIResp openAIChatResponse
	if err := parseJSONResponse(resp, &openAIResp); err != nil {
		return nil, err
	}

	if len(openAIResp.Choices) == 0 {
		return &ChatResponse{}, nil
	}

	choice := openAIResp.Choices[0]
	cr := &ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: Usage{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.TotalTokens,
		},
	}

	if len(choice.Message.ToolCalls) > 0 {
		cr.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			cr.ToolCalls[i] = ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
			}
			cr.ToolCalls[i].Function.Name = tc.Function.Name
			cr.ToolCalls[i].Function.Arguments = tc.Function.Arguments
		}
	}

	return cr, nil
}

func (p *AzureProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	apiKey := p.config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("AZURE_OPENAI_API_KEY")
	}

	messages := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		msg := openAIMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]openAIToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				msg.ToolCalls[j] = openAIToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		messages[i] = msg
	}

	body := map[string]interface{}{
		"messages": messages,
		"stream":   true,
	}

	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}

	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		p.config.BaseURL, p.deployment, p.apiVersion)

	headers := map[string]string{
		"api-key":      apiKey,
		"Content-Type": "application/json",
	}

	httpResp, err := doJSONRequest("POST", url, bytes.NewReader(jsonBody), headers)
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
						Content   string           `json:"content"`
						ToolCalls []openAIToolCall `json:"tool_calls"`
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
				for _, tc := range choice.Delta.ToolCalls {
					toolCall := &ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
					}
					toolCall.Function.Name = tc.Function.Name
					toolCall.Function.Arguments = tc.Function.Arguments
					events <- StreamEvent{
						Type:     StreamEventToolCall,
						ToolCall: toolCall,
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
