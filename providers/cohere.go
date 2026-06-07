package providers

func NewCohereProvider(cfg ProviderConfig, model string) (*OpenAICompatibleProvider, error) {
	return NewOpenAICompatibleProvider(cfg, model, ProviderCohere, "COHERE_API_KEY")
}
