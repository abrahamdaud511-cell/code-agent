package providers

func NewDeepInfraProvider(cfg ProviderConfig, model string) (*OpenAICompatibleProvider, error) {
	return NewOpenAICompatibleProvider(cfg, model, ProviderDeepInfra, "DEEPINFRA_API_KEY")
}
