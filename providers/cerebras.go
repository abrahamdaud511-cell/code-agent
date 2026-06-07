package providers

func NewCerebrasProvider(cfg ProviderConfig, model string) (*OpenAICompatibleProvider, error) {
	return NewOpenAICompatibleProvider(cfg, model, ProviderCerebras, "CEREBRAS_API_KEY")
}
