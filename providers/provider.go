package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"codeagent/config"
)

type ProviderType string

const (
	ProviderOpenAI         ProviderType = "openai"
	ProviderAnthropic      ProviderType = "anthropic"
	ProviderGoogle         ProviderType = "google"
	ProviderGroq           ProviderType = "groq"
	ProviderOllama         ProviderType = "ollama"
	ProviderOpenRouter     ProviderType = "openrouter"
	ProviderGitHubCopilot  ProviderType = "github-copilot"
	ProviderMistral        ProviderType = "mistral"
	ProviderDeepSeek       ProviderType = "deepseek"
	ProviderAWSBedrock     ProviderType = "aws-bedrock"
	ProviderAzure          ProviderType = "azure"
	ProviderCohere         ProviderType = "cohere"
	ProviderPerplexity     ProviderType = "perplexity"
	ProviderXAI            ProviderType = "xai"
	ProviderTogetherAI     ProviderType = "togetherai"
	ProviderDeepInfra      ProviderType = "deepinfra"
	ProviderCerebras       ProviderType = "cerebras"
	ProviderAlibaba        ProviderType = "alibaba"
	ProviderVenice         ProviderType = "venice"
)

type ModelInfo struct {
	Provider    string  `json:"provider"`
	Name        string  `json:"name"`
	InputPrice  float64 `json:"input_price,omitempty"`
	OutputPrice float64 `json:"output_price,omitempty"`
	ContextSize int     `json:"context_size,omitempty"`
}

type Message struct {
	Role       string      `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ToolDefinition struct {
	Type     string     `json:"type"`
	Function FunctionDef `json:"function"`
}

type FunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type Provider interface {
	Name() ProviderType
	Models() ([]ModelInfo, error)
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error)
}

type ChatRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Stream      bool             `json:"stream"`
	TopP        float64          `json:"top_p,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
}

type ChatResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Usage        Usage      `json:"usage"`
	FinishReason string     `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type StreamEvent struct {
	Type     StreamEventType `json:"type"`
	Content  string          `json:"content,omitempty"`
	ToolCall *ToolCall       `json:"tool_call,omitempty"`
	Usage    *Usage          `json:"usage,omitempty"`
	Error    error           `json:"error,omitempty"`
	Done     bool            `json:"done,omitempty"`
}

type StreamEventType string

const (
	StreamEventText     StreamEventType = "text"
	StreamEventToolCall StreamEventType = "tool_call"
	StreamEventUsage    StreamEventType = "usage"
	StreamEventError    StreamEventType = "error"
	StreamEventDone     StreamEventType = "done"
)

type ProviderConfig struct {
	APIKey  string
	BaseURL string
	Models  []string
}

var httpClient = &http.Client{}

func doJSONRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return httpClient.Do(req)
}

func parseJSONResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, target)
}

func LoadCredentials(path string) map[string]string {
	creds := make(map[string]string)
	data, err := os.ReadFile(path)
	if err != nil {
		return creds
	}
	json.Unmarshal(data, &creds)
	return creds
}

func SaveCredentials(path string, creds map[string]string) error {
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(path[:strings.LastIndex(path, string(os.PathSeparator))], 0700)
	return os.WriteFile(path, data, 0600)
}

func NewProvider(pt ProviderType, cfg ProviderConfig, model string) (Provider, error) {
	switch pt {
	case ProviderOpenAI:
		return NewOpenAIProvider(cfg, model)
	case ProviderAnthropic:
		return NewAnthropicProvider(cfg, model)
	case ProviderGoogle:
		return NewGoogleProvider(cfg, model)
	case ProviderGroq:
		return NewGroqProvider(cfg, model)
	case ProviderOllama:
		return NewOllamaProvider(cfg, model)
	case ProviderOpenRouter:
		return NewOpenRouterProvider(cfg, model)
	case ProviderMistral:
		return NewMistralProvider(cfg, model)
	case ProviderDeepSeek:
		return NewDeepSeekProvider(cfg, model)
	case ProviderGitHubCopilot:
		return NewGitHubCopilotProvider(cfg, model)
	case ProviderAWSBedrock:
		return NewAWSBedrockProvider(cfg, model)
	case ProviderAzure:
		return NewAzureProvider(cfg, model)
	case ProviderCohere:
		return NewCohereProvider(cfg, model)
	case ProviderPerplexity:
		return NewPerplexityProvider(cfg, model)
	case ProviderXAI:
		return NewXAIProvider(cfg, model)
	case ProviderTogetherAI:
		return NewTogetherAIProvider(cfg, model)
	case ProviderDeepInfra:
		return NewDeepInfraProvider(cfg, model)
	case ProviderCerebras:
		return NewCerebrasProvider(cfg, model)
	case ProviderAlibaba:
		return NewAlibabaProvider(cfg, model)
	case ProviderVenice:
		return NewVeniceProvider(cfg, model)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", pt)
	}
}

func GetProvider(cfg *config.Config, model string) (Provider, error) {
	if model == "" {
		model = cfg.DefaultModel
	}
	parts := strings.SplitN(model, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid model format: %s (expected provider/model)", model)
	}
	providerName := parts[0]
	modelName := parts[1]

	pCfg, _ := cfg.Providers[providerName]
	apiKey := pCfg.APIKey
	baseURL := pCfg.BaseURL

	if apiKey == "" {
		apiKey = os.Getenv(strings.ToUpper(providerName) + "_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv(strings.ReplaceAll(strings.ToUpper(providerName), "-", "_") + "_API_KEY")
	}

	if providerName == "ollama" || providerName == "github-copilot" {
		apiKey = pCfg.APIKey
	}

	return NewProvider(ProviderType(providerName), ProviderConfig{APIKey: apiKey, BaseURL: baseURL}, modelName)
}
