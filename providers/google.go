package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type GoogleProvider struct {
	config ProviderConfig
	model  string
}

func NewGoogleProvider(cfg ProviderConfig, model string) (*GoogleProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	return &GoogleProvider{
		config: ProviderConfig{
			APIKey:  cfg.APIKey,
			BaseURL: strings.TrimRight(baseURL, "/"),
		},
		model: model,
	}, nil
}

func (p *GoogleProvider) Name() ProviderType {
	return ProviderGoogle
}

func (p *GoogleProvider) Models() ([]ModelInfo, error) {
	return []ModelInfo{
		{Provider: "google", Name: "gemini-2.5-pro"},
		{Provider: "google", Name: "gemini-2.5-flash"},
	}, nil
}

type googleRequest struct {
	Contents         []googleContent `json:"contents"`
	SystemInstruction *googleContent `json:"system_instruction,omitempty"`
	Tools            []googleTool    `json:"tools,omitempty"`
	GenerationConfig struct {
		Temperature float64 `json:"temperature,omitempty"`
		MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
	} `json:"generationConfig"`
}

type googleContent struct {
	Role  string        `json:"role,omitempty"`
	Parts []googlePart  `json:"parts"`
}

type googlePart struct {
	Text       string                `json:"text,omitempty"`
	FunctionCall *googleFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *googleFunctionResponse `json:"functionResponse,omitempty"`
}

type googleFunctionCall struct {
	Name string `json:"name"`
	Args json.RawMessage `json:"args"`
}

type googleFunctionResponse struct {
	Name     string `json:"name"`
	Response struct {
		Content string `json:"content"`
	} `json:"response"`
}

type googleTool struct {
	FunctionDeclarations []googleFunctionDecl `json:"functionDeclarations"`
}

type googleFunctionDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type googleResponse struct {
	Candidates []struct {
		Content googleContent `json:"content"`
		FinishReason string   `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func (p *GoogleProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	gReq := p.toGoogleRequest(req)
	body, err := json.Marshal(gReq)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.config.BaseURL, p.model, p.config.APIKey)
	resp, err := doJSONRequest("POST", url, bytes.NewReader(body),
		map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return nil, err
	}

	var gResp googleResponse
	if err := parseJSONResponse(resp, &gResp); err != nil {
		return nil, err
	}

	return p.toChatResponse(&gResp), nil
}

func (p *GoogleProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	gReq := p.toGoogleRequest(req)
	body, err := json.Marshal(gReq)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse", p.config.BaseURL, p.model, p.config.APIKey)
	resp, err := doJSONRequest("POST", url, bytes.NewReader(body),
		map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return nil, err
	}

	events := make(chan StreamEvent, 100)
	go func() {
		defer resp.Body.Close()
		defer close(events)

		var fullText string
		decoder := json.NewDecoder(resp.Body)
		for decoder.More() {
			var chunk struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
					FinishReason string `json:"finishReason"`
				} `json:"candidates"`
			}
			if err := decoder.Decode(&chunk); err != nil {
				break
			}

			for _, candidate := range chunk.Candidates {
				for _, part := range candidate.Content.Parts {
					if part.Text != "" {
						fullText += part.Text
						events <- StreamEvent{
							Type:    StreamEventText,
							Content: part.Text,
						}
					}
				}
				if candidate.FinishReason != "" {
					events <- StreamEvent{
						Type: StreamEventUsage,
						Usage: &Usage{
							TotalTokens: len(fullText),
						},
					}
					events <- StreamEvent{Type: StreamEventDone, Done: true}
					return
				}
			}
		}
		events <- StreamEvent{Type: StreamEventDone, Done: true}
	}()

	return events, nil
}

func (p *GoogleProvider) toGoogleRequest(req *ChatRequest) *googleRequest {
	gReq := &googleRequest{}

	for _, m := range req.Messages {
		if m.Role == "system" {
			gReq.SystemInstruction = &googleContent{
				Parts: []googlePart{{Text: m.Content}},
			}
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		gReq.Contents = append(gReq.Contents, googleContent{
			Role:  role,
			Parts: []googlePart{{Text: m.Content}},
		})
	}

	if len(req.Tools) > 0 {
		decls := make([]googleFunctionDecl, len(req.Tools))
		for i, t := range req.Tools {
			decls[i] = googleFunctionDecl{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			}
		}
		gReq.Tools = []googleTool{{FunctionDeclarations: decls}}
	}

	gReq.GenerationConfig.Temperature = req.Temperature
	gReq.GenerationConfig.MaxOutputTokens = req.MaxTokens

	return gReq
}

func (p *GoogleProvider) toChatResponse(resp *googleResponse) *ChatResponse {
	cr := &ChatResponse{}

	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		cr.FinishReason = candidate.FinishReason
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				cr.Content += part.Text
			}
		}
	}

	cr.Usage = Usage{
		PromptTokens:     resp.UsageMetadata.PromptTokenCount,
		CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
		TotalTokens:      resp.UsageMetadata.TotalTokenCount,
	}

	return cr
}
