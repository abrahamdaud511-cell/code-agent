package providers

import (
	"strings"
)

type MistralProvider struct {
	*OpenAIProvider
}

func NewMistralProvider(cfg ProviderConfig, model string) (*MistralProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.mistral.ai/v1"
	}
	openAI, err := NewOpenAIProvider(ProviderConfig{
		APIKey:  cfg.APIKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
	}, model)
	if err != nil {
		return nil, err
	}
	return &MistralProvider{openAI}, nil
}

func (p *MistralProvider) Name() ProviderType {
	return ProviderMistral
}

func (p *MistralProvider) Models() ([]ModelInfo, error) {
	return []ModelInfo{
		{Provider: "mistral", Name: "mistral-large-2505"},
		{Provider: "mistral", Name: "mistral-saba"},
		{Provider: "mistral", Name: "codestral"},
	}, nil
}
