package providers

import (
	"strings"
)

type OpenRouterProvider struct {
	*OpenAIProvider
}

func NewOpenRouterProvider(cfg ProviderConfig, model string) (*OpenRouterProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	openAI, err := NewOpenAIProvider(ProviderConfig{
		APIKey:  cfg.APIKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
	}, model)
	if err != nil {
		return nil, err
	}
	return &OpenRouterProvider{openAI}, nil
}

func (p *OpenRouterProvider) Name() ProviderType {
	return ProviderOpenRouter
}

func (p *OpenRouterProvider) Models() ([]ModelInfo, error) {
	resp, err := doJSONRequest("GET", p.config.BaseURL+"/models", nil,
		map[string]string{
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
			Provider: "openrouter",
			Name:     m.ID,
		})
	}
	return models, nil
}
