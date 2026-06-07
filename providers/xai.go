package providers

func NewXAIProvider(cfg ProviderConfig, model string) (*OpenAICompatibleProvider, error) {
	return NewOpenAICompatibleProvider(cfg, model, ProviderXAI, "XAI_API_KEY")
}
