package providers

func NewAlibabaProvider(cfg ProviderConfig, model string) (*OpenAICompatibleProvider, error) {
	return NewOpenAICompatibleProvider(cfg, model, ProviderAlibaba, "ALIBABA_API_KEY")
}
