package providers

func NewTogetherAIProvider(cfg ProviderConfig, model string) (*OpenAICompatibleProvider, error) {
	return NewOpenAICompatibleProvider(cfg, model, ProviderTogetherAI, "TOGETHERAI_API_KEY")
}
