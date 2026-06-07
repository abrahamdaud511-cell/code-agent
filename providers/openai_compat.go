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

type OpenAICompatibleProvider struct {
	config    ProviderConfig
	model     string
	provider  ProviderType
	apiKeyEnv string
}

func NewOpenAICompatibleProvider(cfg ProviderConfig, model string, pt ProviderType, apiKeyEnv string) (*OpenAICompatibleProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		switch pt {
		case ProviderCohere:
			baseURL = "https://api.cohere.com/v1"
		case ProviderPerplexity:
			baseURL = "https://api.perplexity.ai"
		case ProviderXAI:
			baseURL = "https://api.x.ai/v1"
		case ProviderTogetherAI:
			baseURL = "https://api.together.xyz/v1"
		case ProviderDeepInfra:
			baseURL = "https://api.deepinfra.com/v1/openai"
		case ProviderCerebras:
			baseURL = "https://api.cerebras.ai/v1"
		case ProviderAlibaba:
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		case ProviderVenice:
			baseURL = "https://api.venice.ai/api/v1"
		default:
			baseURL = "https://api.openai.com/v1"
		}
	}

	return &OpenAICompatibleProvider{
		config: ProviderConfig{
			APIKey:  cfg.APIKey,
			BaseURL: strings.TrimRight(baseURL, "/"),
		},
		model:     model,
		provider:  pt,
		apiKeyEnv: apiKeyEnv,
	}, nil
}

func (p *OpenAICompatibleProvider) Name() ProviderType {
	return p.provider
}

func (p *OpenAICompatibleProvider) Models() ([]ModelInfo, error) {
	switch p.provider {
	case ProviderCohere:
		return []ModelInfo{
			{Provider: "cohere", Name: "command-r-plus", ContextSize: 128000},
			{Provider: "cohere", Name: "command-r", ContextSize: 128000},
			{Provider: "cohere", Name: "command-a", ContextSize: 128000},
		}, nil
	case ProviderPerplexity:
		return []ModelInfo{
			{Provider: "perplexity", Name: "sonar-pro", ContextSize: 200000},
			{Provider: "perplexity", Name: "sonar", ContextSize: 127000},
		}, nil
	case ProviderXAI:
		return []ModelInfo{
			{Provider: "xai", Name: "grok-3", ContextSize: 131072},
			{Provider: "xai", Name: "grok-3-mini", ContextSize: 131072},
			{Provider: "xai", Name: "grok-3-latest", ContextSize: 131072},
		}, nil
	case ProviderTogetherAI:
		return []ModelInfo{
			{Provider: "togetherai", Name: "llama-4-17b", ContextSize: 128000},
			{Provider: "togetherai", Name: "deepseek-v3", ContextSize: 64000},
			{Provider: "togetherai", Name: "qwen-2.5-coder", ContextSize: 32768},
		}, nil
	case ProviderDeepInfra:
		return []ModelInfo{
			{Provider: "deepinfra", Name: "llama-4-scout", ContextSize: 128000},
			{Provider: "deepinfra", Name: "deepseek-v3", ContextSize: 128000},
			{Provider: "deepinfra", Name: "qwen-2.5-coder", ContextSize: 32768},
		}, nil
	case ProviderCerebras:
		return []ModelInfo{
			{Provider: "cerebras", Name: "llama-4-scout", ContextSize: 128000},
		}, nil
	case ProviderAlibaba:
		return []ModelInfo{
			{Provider: "alibaba", Name: "qwen-max", ContextSize: 32768},
			{Provider: "alibaba", Name: "qwen-plus", ContextSize: 131072},
			{Provider: "alibaba", Name: "qwen-turbo", ContextSize: 131072},
		}, nil
	case ProviderVenice:
		return []ModelInfo{
			{Provider: "venice", Name: "llama-4-scout", ContextSize: 128000},
			{Provider: "venice", Name: "deepseek-r1", ContextSize: 128000},
		}, nil
	default:
		return []ModelInfo{
			{Provider: string(p.provider), Name: p.model},
		}, nil
	}
}

func (p *OpenAICompatibleProvider) getAPIKey() string {
	if p.config.APIKey != "" {
		return p.config.APIKey
	}
	if p.apiKeyEnv != "" {
		return os.Getenv(p.apiKeyEnv)
	}
	return os.Getenv(strings.ToUpper(string(p.provider)) + "_API_KEY")
}

func (p *OpenAICompatibleProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	apiKey := p.getAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("no API key configured for %s. Set %s_API_KEY environment variable", p.provider, strings.ToUpper(string(p.provider)))
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

	model := p.model
	if req.Model != "" {
		model = req.Model
	}

	body := map[string]interface{}{
		"model":    model,
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
	if req.TopP > 0 {
		body["top_p"] = req.TopP
	}

	jsonBody, _ := json.Marshal(body)

	headers := map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}

	resp, err := doJSONRequest("POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(jsonBody), headers)
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

func (p *OpenAICompatibleProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	apiKey := p.getAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("no API key configured for %s", p.provider)
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

	model := p.model
	if req.Model != "" {
		model = req.Model
	}

	body := map[string]interface{}{
		"model":    model,
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

	headers := map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}

	httpResp, err := doJSONRequest("POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(jsonBody), headers)
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
