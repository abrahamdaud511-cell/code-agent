package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
)

type OpenAIProvider struct {
	config ProviderConfig
	model  string
}

func NewOpenAIProvider(cfg ProviderConfig, model string) (*OpenAIProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		config: ProviderConfig{
			APIKey:  cfg.APIKey,
			BaseURL: strings.TrimRight(baseURL, "/"),
		},
		model: model,
	}, nil
}

func (p *OpenAIProvider) Name() ProviderType {
	return ProviderOpenAI
}

func (p *OpenAIProvider) Models() ([]ModelInfo, error) {
	resp, err := doJSONRequest("GET", p.config.BaseURL+"/models", nil, map[string]string{
		"Authorization": "Bearer " + p.config.APIKey,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := parseJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0)
	for _, m := range result.Data {
		models = append(models, ModelInfo{
			Provider: "openai",
			Name:     m.ID,
		})
	}
	return models, nil
}

type openAIChatRequest struct {
	Model       string           `json:"model"`
	Messages    []openAIMessage  `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Stream      bool             `json:"stream"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	openAIReq := p.toOpenAIRequest(req)
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, err
	}

	resp, err := doJSONRequest("POST", p.config.BaseURL+"/chat/completions",
		bytes.NewReader(body),
		map[string]string{
			"Authorization": "Bearer " + p.config.APIKey,
			"Content-Type":  "application/json",
		})
	if err != nil {
		return nil, err
	}

	var openAIResp openAIChatResponse
	if err := parseJSONResponse(resp, &openAIResp); err != nil {
		return nil, err
	}

	return p.toChatResponse(&openAIResp), nil
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	openAIReq := p.toOpenAIRequest(req)
	openAIReq.Stream = true
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, err
	}

	httpResp, err := doJSONRequest("POST", p.config.BaseURL+"/chat/completions",
		bytes.NewReader(body),
		map[string]string{
			"Authorization": "Bearer " + p.config.APIKey,
			"Content-Type":  "application/json",
		})
	if err != nil {
		return nil, err
	}

	events := make(chan StreamEvent, 100)
	go p.processStream(httpResp.Body, events)
	return events, nil
}

func (p *OpenAIProvider) processStream(body io.ReadCloser, events chan<- StreamEvent) {
	defer body.Close()
	defer close(events)

	scanner := bufio.NewScanner(body)
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
			Usage *Usage `json:"usage,omitempty"`
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
				if chunk.Usage != nil {
					events <- StreamEvent{
						Type:  StreamEventUsage,
						Usage: chunk.Usage,
					}
				}
				events <- StreamEvent{Type: StreamEventDone, Done: true}
				return
			}
		}
	}
}

func (p *OpenAIProvider) toOpenAIRequest(req *ChatRequest) *openAIChatRequest {
	model := p.model
	if req.Model != "" {
		model = req.Model
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

	return &openAIChatRequest{
		Model:       model,
		Messages:    messages,
		Tools:       req.Tools,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
}

func (p *OpenAIProvider) toChatResponse(resp *openAIChatResponse) *ChatResponse {
	if len(resp.Choices) == 0 {
		return &ChatResponse{}
	}

	choice := resp.Choices[0]
	cr := &ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
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

	return cr
}
