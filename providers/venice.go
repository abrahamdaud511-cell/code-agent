package providers

func NewVeniceProvider(cfg ProviderConfig, model string) (*OpenAICompatibleProvider, error) {
	return NewOpenAICompatibleProvider(cfg, model, ProviderVenice, "VENICE_API_KEY")
}
