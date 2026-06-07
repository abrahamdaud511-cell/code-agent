package providers

import (
	"strings"
)

type DeepSeekProvider struct {
	*OpenAIProvider
}

func NewDeepSeekProvider(cfg ProviderConfig, model string) (*DeepSeekProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	openAI, err := NewOpenAIProvider(ProviderConfig{
		APIKey:  cfg.APIKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
	}, model)
	if err != nil {
		return nil, err
	}
	return &DeepSeekProvider{openAI}, nil
}

func (p *DeepSeekProvider) Name() ProviderType {
	return ProviderDeepSeek
}

func (p *DeepSeekProvider) Models() ([]ModelInfo, error) {
	return []ModelInfo{
		{Provider: "deepseek", Name: "deepseek-chat"},
		{Provider: "deepseek", Name: "deepseek-reasoner"},
	}, nil
}
