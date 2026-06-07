package providers

func NewPerplexityProvider(cfg ProviderConfig, model string) (*OpenAICompatibleProvider, error) {
	return NewOpenAICompatibleProvider(cfg, model, ProviderPerplexity, "PERPLEXITY_API_KEY")
}
