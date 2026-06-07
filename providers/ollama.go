package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
)

type OllamaProvider struct {
	config ProviderConfig
	model  string
}

func NewOllamaProvider(cfg ProviderConfig, model string) (*OllamaProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		config: ProviderConfig{
			BaseURL: strings.TrimRight(baseURL, "/"),
		},
		model: model,
	}, nil
}

func (p *OllamaProvider) Name() ProviderType {
	return ProviderOllama
}

func (p *OllamaProvider) Models() ([]ModelInfo, error) {
	resp, err := doJSONRequest("GET", p.config.BaseURL+"/api/tags", nil, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := parseJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0)
	for _, m := range result.Models {
		models = append(models, ModelInfo{
			Provider: "ollama",
			Name:     m.Name,
		})
	}
	return models, nil
}

type ollamaRequest struct {
	Model    string      `json:"model"`
	Messages []ollamaMsg `json:"messages"`
	Stream   bool        `json:"stream"`
	Options  struct {
		Temperature float64 `json:"temperature,omitempty"`
	} `json:"options,omitempty"`
}

type ollamaMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

func (p *OllamaProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	oReq := p.toOllamaRequest(req)
	oReq.Stream = false
	body, err := json.Marshal(oReq)
	if err != nil {
		return nil, err
	}

	resp, err := doJSONRequest("POST", p.config.BaseURL+"/api/chat",
		bytes.NewReader(body),
		map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return nil, err
	}

	var oResp ollamaResponse
	if err := parseJSONResponse(resp, &oResp); err != nil {
		return nil, err
	}

	return &ChatResponse{
		Content: oResp.Message.Content,
	}, nil
}

func (p *OllamaProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	oReq := p.toOllamaRequest(req)
	oReq.Stream = true
	body, err := json.Marshal(oReq)
	if err != nil {
		return nil, err
	}

	httpResp, err := doJSONRequest("POST", p.config.BaseURL+"/api/chat",
		bytes.NewReader(body),
		map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return nil, err
	}

	events := make(chan StreamEvent, 100)
	go func() {
		defer httpResp.Body.Close()
		defer close(events)

		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			var chunk ollamaResponse
			if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
				continue
			}
			if chunk.Message.Content != "" {
				events <- StreamEvent{
					Type:    StreamEventText,
					Content: chunk.Message.Content,
				}
			}
			if chunk.Done {
				events <- StreamEvent{Type: StreamEventDone, Done: true}
				return
			}
		}
		events <- StreamEvent{Type: StreamEventDone, Done: true}
	}()

	return events, nil
}

func (p *OllamaProvider) toOllamaRequest(req *ChatRequest) *ollamaRequest {
	model := p.model
	if req.Model != "" {
		model = req.Model
	}

	messages := make([]ollamaMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = ollamaMsg{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	oReq := &ollamaRequest{
		Model:    model,
		Messages: messages,
	}
	oReq.Options.Temperature = req.Temperature

	return oReq
}
