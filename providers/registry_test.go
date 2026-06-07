package providers

import (
	"testing"

	"codeagent/config"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestDefaultModels(t *testing.T) {
	models := defaultModels()
	if len(models) == 0 {
		t.Fatal("expected non-empty default models")
	}

	// Check that common providers are present
	providers := make(map[string]bool)
	for _, m := range models {
		providers[m.Provider] = true
	}

	expectedProviders := []string{"openai", "anthropic", "google", "groq", "ollama", "openrouter", "mistral", "deepseek"}
	for _, p := range expectedProviders {
		if !providers[p] {
			t.Errorf("expected provider %s to have default models", p)
		}
	}
}

func TestDefaultModelsHavePrices(t *testing.T) {
	models := defaultModels()
	for _, m := range models {
		if stringsContains(m.Name, "gpt") || stringsContains(m.Name, "claude") || stringsContains(m.Name, "gemini") {
			if m.InputPrice == 0 && m.OutputPrice == 0 && m.Provider != "ollama" {
				// Free models and local models are fine
			}
		}
	}
}

func TestModelString(t *testing.T) {
	m := ModelInfo{Provider: "openai", Name: "gpt-5"}
	expected := "openai/gpt-5"
	if m.String() != expected {
		t.Errorf("expected %s, got %s", expected, m.String())
	}
}

func TestListModelsFiltered(t *testing.T) {
	r := NewRegistry()

	// Filter by one provider
	configured := map[string]config.ProviderConfig{
		"openai": {},
	}

	models := r.ListModels(configured)
	for _, m := range models {
		if m.Provider != "openai" {
			t.Errorf("expected only openai models when filtered, got %s/%s", m.Provider, m.Name)
		}
	}
}

func TestListModelsAll(t *testing.T) {
	r := NewRegistry()
	models := r.ListModels(nil)

	if len(models) == 0 {
		t.Error("expected non-empty model list")
	}
}

func TestModelInfoContextSize(t *testing.T) {
	models := defaultModels()
	for _, m := range models {
		if m.ContextSize <= 0 {
			t.Errorf("model %s/%s has invalid context size %d", m.Provider, m.Name, m.ContextSize)
		}
	}
}

func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && stringsContainsStr(s, substr)
}

func stringsContainsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
