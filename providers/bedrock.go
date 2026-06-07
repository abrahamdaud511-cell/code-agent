package providers

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type AWSBedrockProvider struct {
	config    ProviderConfig
	model     string
	region    string
	accessKey string
	secretKey string
}

func NewAWSBedrockProvider(cfg ProviderConfig, model string) (*AWSBedrockProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://bedrock-runtime.us-east-1.amazonaws.com"
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = "us-east-1"
	}

	return &AWSBedrockProvider{
		config: ProviderConfig{
			APIKey:  cfg.APIKey,
			BaseURL: strings.TrimRight(baseURL, "/"),
		},
		model:     model,
		region:    region,
		accessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
		secretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}, nil
}

func (p *AWSBedrockProvider) Name() ProviderType {
	return ProviderAWSBedrock
}

func (p *AWSBedrockProvider) Models() ([]ModelInfo, error) {
	return []ModelInfo{
		{Provider: "aws-bedrock", Name: "claude-sonnet-4", ContextSize: 200000},
		{Provider: "aws-bedrock", Name: "claude-haiku-4", ContextSize: 200000},
		{Provider: "aws-bedrock", Name: "claude-opus-4", ContextSize: 200000},
		{Provider: "aws-bedrock", Name: "llama-4", ContextSize: 128000},
		{Provider: "aws-bedrock", Name: "mistral-large", ContextSize: 128000},
	}, nil
}

func (p *AWSBedrockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	model := p.model
	if req.Model != "" {
		model = req.Model
	}

	modelID := p.getModelID(model)

	var messages []map[string]interface{}
	var system string
	for _, m := range req.Messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		messages = append(messages, map[string]interface{}{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	body := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        4096,
		"messages":          messages,
	}

	if system != "" {
		body["system"] = system
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}

	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/model/%s/invoke", p.config.BaseURL, modelID)
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}

	if p.accessKey != "" && p.secretKey != "" {
		p.signRequest("POST", url, jsonBody, headers)
	}

	resp, err := doJSONRequest("POST", url, bytes.NewReader(jsonBody), headers)
	if err != nil {
		return nil, err
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := parseJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	cr := &ChatResponse{
		FinishReason: result.StopReason,
		Usage: Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}

	for _, block := range result.Content {
		if block.Type == "text" {
			cr.Content += block.Text
		}
	}

	return cr, nil
}

func (p *AWSBedrockProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error) {
	model := p.model
	if req.Model != "" {
		model = req.Model
	}

	modelID := p.getModelID(model)

	var messages []map[string]interface{}
	var system string
	for _, m := range req.Messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		messages = append(messages, map[string]interface{}{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	body := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        4096,
		"messages":          messages,
	}

	if system != "" {
		body["system"] = system
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}

	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/model/%s/invoke-with-response-stream", p.config.BaseURL, modelID)
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}

	if p.accessKey != "" && p.secretKey != "" {
		p.signRequest("POST", url, jsonBody, headers)
	}

	httpResp, err := doJSONRequest("POST", url, bytes.NewReader(jsonBody), headers)
	if err != nil {
		return nil, err
	}

	events := make(chan StreamEvent, 100)
	go func() {
		defer httpResp.Body.Close()
		defer close(events)

		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			var event struct {
				Type  string `json:"type"`
				Delta struct {
					Text string `json:"text"`
				} `json:"delta"`
			}

			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Text != "" {
					events <- StreamEvent{
						Type:    StreamEventText,
						Content: event.Delta.Text,
					}
				}
			case "message_stop":
				events <- StreamEvent{Type: StreamEventDone, Done: true}
				return
			}
		}
	}()

	return events, nil
}

func (p *AWSBedrockProvider) getModelID(model string) string {
	switch {
	case strings.Contains(model, "claude-sonnet"):
		return "anthropic.claude-sonnet-4-20250514"
	case strings.Contains(model, "claude-haiku"):
		return "anthropic.claude-haiku-4-20250514"
	case strings.Contains(model, "claude-opus"):
		return "anthropic.claude-opus-4-20250514"
	case strings.Contains(model, "llama"):
		return "meta.llama-4-20250417"
	case strings.Contains(model, "mistral-large"):
		return "mistral.mistral-large-2505"
	default:
		return "anthropic.claude-sonnet-4-20250514"
	}
}

func (p *AWSBedrockProvider) signRequest(method, url string, body []byte, headers map[string]string) {
	// Simplified AWS SigV4 signing
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStr := now.Format("20060102")

	headers["X-Amz-Date"] = amzDate
	headers["Host"] = "bedrock-runtime." + p.region + ".amazonaws.com"

	payloadHash := sha256Hex(body)
	headers["X-Amz-Content-SHA256"] = payloadHash

	canonicalHeaders := "content-type:application/json\nhost:" + headers["Host"] + "\nx-amz-content-sha256:" + payloadHash + "\nx-amz-date:" + amzDate + "\n"
	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := method + "\n" + "/\n" + "\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + payloadHash

	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := dateStr + "/" + p.region + "/bedrock/aws4_request"
	stringToSign := algorithm + "\n" + amzDate + "\n" + credentialScope + "\n" + sha256Hex([]byte(canonicalRequest))

	signingKey := p.getSignatureKey(p.secretKey, dateStr, p.region, "bedrock")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	headers["Authorization"] = fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, p.accessKey, credentialScope, signedHeaders, signature)
}

func (p *AWSBedrockProvider) getSignatureKey(key, dateStamp, regionName, serviceName string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+key), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(regionName))
	kService := hmacSHA256(kRegion, []byte(serviceName))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func hmacSHA256(key []byte, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
