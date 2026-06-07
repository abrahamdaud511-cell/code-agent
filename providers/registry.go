package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"codeagent/config"
)

type ModelRegistry struct {
	mu     sync.RWMutex
	models []ModelInfo
	cache  *ModelCache
}

type ModelCache struct {
	Models    []ModelInfo `json:"models"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func NewRegistry() *ModelRegistry {
	return &ModelRegistry{
		models: defaultModels(),
	}
}

func (r *ModelRegistry) Refresh() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	allModels := defaultModels()
	providers := []ProviderType{
		ProviderOpenAI, ProviderAnthropic, ProviderGoogle,
		ProviderGroq, ProviderOllama, ProviderOpenRouter,
		ProviderMistral, ProviderDeepSeek,
		ProviderGitHubCopilot, ProviderAWSBedrock, ProviderAzure,
		ProviderCohere, ProviderPerplexity, ProviderXAI,
		ProviderTogetherAI, ProviderDeepInfra, ProviderCerebras,
		ProviderAlibaba, ProviderVenice,
	}

	for _, pt := range providers {
		prov, err := NewProvider(pt, ProviderConfig{}, "")
		if err != nil {
			continue
		}
		models, err := prov.Models()
		if err == nil {
			allModels = append(allModels, models...)
		}
	}

	r.models = allModels
	return nil
}

func (r *ModelRegistry) ListModels(configuredProviders map[string]config.ProviderConfig) []ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(configuredProviders) == 0 {
		return r.models
	}

	filtered := make([]ModelInfo, 0)
	for _, m := range r.models {
		if _, ok := configuredProviders[m.Provider]; ok {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) == 0 {
		return r.models
	}
	return filtered
}

func defaultModels() []ModelInfo {
	return []ModelInfo{
		// OpenAI
		{Provider: "openai", Name: "gpt-5", InputPrice: 15.0, OutputPrice: 60.0, ContextSize: 128000},
		{Provider: "openai", Name: "gpt-5-turbo", InputPrice: 10.0, OutputPrice: 40.0, ContextSize: 128000},
		{Provider: "openai", Name: "gpt-4o", InputPrice: 2.5, OutputPrice: 10.0, ContextSize: 128000},
		{Provider: "openai", Name: "gpt-4o-mini", InputPrice: 0.15, OutputPrice: 0.6, ContextSize: 128000},
		{Provider: "openai", Name: "o4", InputPrice: 10.0, OutputPrice: 40.0, ContextSize: 200000},
		{Provider: "openai", Name: "o4-mini", InputPrice: 1.1, OutputPrice: 4.4, ContextSize: 200000},
		{Provider: "openai", Name: "o3", InputPrice: 10.0, OutputPrice: 40.0, ContextSize: 200000},
		{Provider: "openai", Name: "o3-mini", InputPrice: 1.1, OutputPrice: 4.4, ContextSize: 200000},

		// Anthropic
		{Provider: "anthropic", Name: "claude-sonnet-4", InputPrice: 3.0, OutputPrice: 15.0, ContextSize: 200000},
		{Provider: "anthropic", Name: "claude-haiku-4", InputPrice: 1.0, OutputPrice: 5.0, ContextSize: 200000},
		{Provider: "anthropic", Name: "claude-opus-4", InputPrice: 15.0, OutputPrice: 75.0, ContextSize: 200000},

		// Google
		{Provider: "google", Name: "gemini-2.5-pro", InputPrice: 1.25, OutputPrice: 10.0, ContextSize: 1048576},
		{Provider: "google", Name: "gemini-2.5-flash", InputPrice: 0.15, OutputPrice: 0.6, ContextSize: 1048576},

		// Groq
		{Provider: "groq", Name: "llama-4-scout", InputPrice: 0.1, OutputPrice: 0.4, ContextSize: 128000},
		{Provider: "groq", Name: "llama-4-maverick", InputPrice: 0.2, OutputPrice: 0.8, ContextSize: 128000},
		{Provider: "groq", Name: "deepseek-r1-distill", InputPrice: 0.1, OutputPrice: 0.4, ContextSize: 128000},

		// OpenRouter
		{Provider: "openrouter", Name: "auto", InputPrice: 0, OutputPrice: 0, ContextSize: 128000},

		// Mistral
		{Provider: "mistral", Name: "mistral-large-2505", InputPrice: 2.0, OutputPrice: 6.0, ContextSize: 128000},
		{Provider: "mistral", Name: "mistral-saba", InputPrice: 0.2, OutputPrice: 0.6, ContextSize: 128000},
		{Provider: "mistral", Name: "codestral", InputPrice: 1.0, OutputPrice: 3.0, ContextSize: 256000},

		// DeepSeek
		{Provider: "deepseek", Name: "deepseek-chat", InputPrice: 0.27, OutputPrice: 1.1, ContextSize: 128000},
		{Provider: "deepseek", Name: "deepseek-reasoner", InputPrice: 0.55, OutputPrice: 2.19, ContextSize: 128000},

		// Ollama (local)
		{Provider: "ollama", Name: "llama4", InputPrice: 0, OutputPrice: 0, ContextSize: 128000},
		{Provider: "ollama", Name: "deepseek-r1", InputPrice: 0, OutputPrice: 0, ContextSize: 128000},
		{Provider: "ollama", Name: "mistral", InputPrice: 0, OutputPrice: 0, ContextSize: 32768},
		{Provider: "ollama", Name: "codellama", InputPrice: 0, OutputPrice: 0, ContextSize: 16384},
		{Provider: "ollama", Name: "qwen2.5-coder", InputPrice: 0, OutputPrice: 0, ContextSize: 32768},
		{Provider: "ollama", Name: "phi-4", InputPrice: 0, OutputPrice: 0, ContextSize: 16384},

		// GitHub Copilot
		{Provider: "github-copilot", Name: "gpt-4o-copilot", InputPrice: 0, OutputPrice: 0, ContextSize: 128000},
		{Provider: "github-copilot", Name: "claude-sonnet-4-copilot", InputPrice: 0, OutputPrice: 0, ContextSize: 200000},

		// AWS Bedrock
		{Provider: "aws-bedrock", Name: "claude-sonnet-4", InputPrice: 3.0, OutputPrice: 15.0, ContextSize: 200000},
		{Provider: "aws-bedrock", Name: "claude-haiku-4", InputPrice: 1.0, OutputPrice: 5.0, ContextSize: 200000},
		{Provider: "aws-bedrock", Name: "claude-opus-4", InputPrice: 15.0, OutputPrice: 75.0, ContextSize: 200000},
		{Provider: "aws-bedrock", Name: "llama-4", InputPrice: 0.5, OutputPrice: 1.5, ContextSize: 128000},

		// Azure
		{Provider: "azure", Name: "gpt-4o", InputPrice: 2.5, OutputPrice: 10.0, ContextSize: 128000},
		{Provider: "azure", Name: "gpt-4o-mini", InputPrice: 0.15, OutputPrice: 0.6, ContextSize: 128000},
		{Provider: "azure", Name: "gpt-5", InputPrice: 15.0, OutputPrice: 60.0, ContextSize: 128000},

		// Cohere
		{Provider: "cohere", Name: "command-r-plus", InputPrice: 2.5, OutputPrice: 10.0, ContextSize: 128000},
		{Provider: "cohere", Name: "command-r", InputPrice: 0.5, OutputPrice: 1.5, ContextSize: 128000},
		{Provider: "cohere", Name: "command-a", InputPrice: 2.5, OutputPrice: 10.0, ContextSize: 128000},

		// Perplexity
		{Provider: "perplexity", Name: "sonar-pro", InputPrice: 3.0, OutputPrice: 15.0, ContextSize: 200000},
		{Provider: "perplexity", Name: "sonar", InputPrice: 1.0, OutputPrice: 5.0, ContextSize: 127000},

		// xAI
		{Provider: "xai", Name: "grok-3", InputPrice: 5.0, OutputPrice: 15.0, ContextSize: 131072},
		{Provider: "xai", Name: "grok-3-mini", InputPrice: 2.0, OutputPrice: 8.0, ContextSize: 131072},

		// TogetherAI
		{Provider: "togetherai", Name: "llama-4-17b", InputPrice: 0.2, OutputPrice: 0.8, ContextSize: 128000},
		{Provider: "togetherai", Name: "deepseek-v3", InputPrice: 1.0, OutputPrice: 3.0, ContextSize: 64000},
		{Provider: "togetherai", Name: "qwen-2.5-coder", InputPrice: 0.3, OutputPrice: 0.9, ContextSize: 32768},

		// DeepInfra
		{Provider: "deepinfra", Name: "llama-4-scout", InputPrice: 0.2, OutputPrice: 0.8, ContextSize: 128000},
		{Provider: "deepinfra", Name: "deepseek-v3", InputPrice: 0.5, OutputPrice: 1.5, ContextSize: 128000},

		// Cerebras
		{Provider: "cerebras", Name: "llama-4-scout", InputPrice: 0.1, OutputPrice: 0.4, ContextSize: 128000},

		// Alibaba/Qwen
		{Provider: "alibaba", Name: "qwen-max", InputPrice: 1.6, OutputPrice: 6.4, ContextSize: 32768},
		{Provider: "alibaba", Name: "qwen-plus", InputPrice: 0.8, OutputPrice: 3.2, ContextSize: 131072},
		{Provider: "alibaba", Name: "qwen-turbo", InputPrice: 0.3, OutputPrice: 1.2, ContextSize: 131072},

		// Venice
		{Provider: "venice", Name: "llama-4-scout", InputPrice: 0, OutputPrice: 0, ContextSize: 128000},
		{Provider: "venice", Name: "deepseek-r1", InputPrice: 0, OutputPrice: 0, ContextSize: 128000},
	}
}

func loadCache(cacheDir string) *ModelCache {
	cacheFile := filepath.Join(cacheDir, "models-cache.json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}
	var cache ModelCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}
	return &cache
}

func saveCache(cacheDir string, cache *ModelCache) error {
	os.MkdirAll(cacheDir, 0755)
	cacheFile := filepath.Join(cacheDir, "models-cache.json")
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cacheFile, data, 0644)
}

func (m ModelInfo) String() string {
	return fmt.Sprintf("%s/%s", m.Provider, m.Name)
}
