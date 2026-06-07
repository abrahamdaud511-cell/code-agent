package providers

import (
	"strings"
)

type GroqProvider struct {
	*OpenAIProvider
}

func NewGroqProvider(cfg ProviderConfig, model string) (*GroqProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}
	openAI, err := NewOpenAIProvider(ProviderConfig{
		APIKey:  cfg.APIKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
	}, model)
	if err != nil {
		return nil, err
	}
	return &GroqProvider{openAI}, nil
}

func (p *GroqProvider) Name() ProviderType {
	return ProviderGroq
}

func (p *GroqProvider) Models() ([]ModelInfo, error) {
	return []ModelInfo{
		{Provider: "groq", Name: "llama-4-scout"},
		{Provider: "groq", Name: "llama-4-maverick"},
		{Provider: "groq", Name: "deepseek-r1-distill"},
		{Provider: "groq", Name: "mixtral-8x7b"},
		{Provider: "groq", Name: "gemma2-9b"},
	}, nil
}
