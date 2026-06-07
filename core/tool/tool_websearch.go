package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type WebSearchTool struct{}

type WebSearchArgs struct {
	Query    string `json:"query"`
	Limit    int    `json:"limit,omitempty"`
	Provider string `json:"provider,omitempty"`
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{}
}

func (t *WebSearchTool) Name() string {
	return "websearch"
}

func (t *WebSearchTool) Description() string {
	return "Search the web for information. Uses Google Custom Search, Bing Search, or DuckDuckGo. Returns relevant snippets and URLs."
}

func (t *WebSearchTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results to return (default: 8, max: 20)",
			},
			"provider": map[string]interface{}{
				"type":        "string",
				"description": "Search provider: google, bing, duckduckgo (default: auto)",
				"enum":        []string{"auto", "google", "bing", "duckduckgo"},
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args WebSearchArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}

	results, err := t.search(ctx, args.Query, args.Provider, limit)
	if err != nil {
		// Fallback: try DuckDuckGo if primary fails
		results, err = t.searchDuckDuckGo(ctx, args.Query, limit)
		if err != nil {
			return "", fmt.Errorf("search failed: %w", err)
		}
	}

	if len(results) == 0 {
		return fmt.Sprintf("No results found for: %s", args.Query), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for \"%s\":\n\n", args.Query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", r.URL))
		if r.Content != "" {
			content := r.Content
			if len(content) > 300 {
				content = content[:300] + "..."
			}
			sb.WriteString(fmt.Sprintf("   %s\n", content))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func (t *WebSearchTool) search(ctx context.Context, query, provider string, limit int) ([]SearchResult, error) {
	switch provider {
	case "google":
		return t.searchGoogle(ctx, query, limit)
	case "bing":
		return t.searchBing(ctx, query, limit)
	case "duckduckgo":
		return t.searchDuckDuckGo(ctx, query, limit)
	default:
		return t.searchDuckDuckGo(ctx, query, limit)
	}
}

func (t *WebSearchTool) searchGoogle(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	apiKey := os.Getenv("GOOGLE_CSE_API_KEY")
	cseID := os.Getenv("GOOGLE_CSE_ID")
	if apiKey == "" || cseID == "" {
		return nil, fmt.Errorf("GOOGLE_CSE_API_KEY and GOOGLE_CSE_ID required")
	}

	u := fmt.Sprintf("https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&num=%d",
		url.QueryEscape(apiKey),
		url.QueryEscape(cseID),
		url.QueryEscape(query),
		limit,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(result.Items))
	for i, item := range result.Items {
		results[i] = SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Content: item.Snippet,
		}
	}

	return results, nil
}

func (t *WebSearchTool) searchBing(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	apiKey := os.Getenv("BING_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("BING_API_KEY required")
	}

	u := fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?q=%s&count=%d",
		url.QueryEscape(query),
		limit,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		WebPages struct {
			Value []struct {
				Name    string `json:"name"`
				URL     string `json:"url"`
				Snippet string `json:"snippet"`
			} `json:"value"`
		} `json:"webPages"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(result.WebPages.Value))
	for i, item := range result.WebPages.Value {
		results[i] = SearchResult{
			Title:   item.Name,
			URL:     item.URL,
			Content: item.Snippet,
		}
	}

	return results, nil
}

func (t *WebSearchTool) searchDuckDuckGo(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	u := fmt.Sprintf("https://lite.duckduckgo.com/lite/?q=%s", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CodeAgent/1.0")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseDuckDuckGoHTML(string(body), limit), nil
}

func parseDuckDuckGoHTML(html string, limit int) []SearchResult {
	var results []SearchResult
	lines := strings.Split(html, "\n")
	inResult := false
	var current SearchResult

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "class=\"result-link\"") || strings.Contains(trimmed, "class=\"result__a\"") {
			inResult = true
			// Extract URL and title from <a> tag
			if start := strings.Index(trimmed, "href=\""); start >= 0 {
				start += 6
				if end := strings.Index(trimmed[start:], "\""); end >= 0 {
					current.URL = trimmed[start : start+end]
				}
			}
			if start := strings.Index(trimmed, ">"); start >= 0 {
				start++
				if end := strings.LastIndex(trimmed, "</a>"); end >= 0 {
					current.Title = strings.TrimSpace(trimmed[start:end])
				}
			}
		} else if strings.Contains(trimmed, "class=\"result-snippet\"") || strings.Contains(trimmed, "class=\"result__snippet\"") {
			if start := strings.Index(trimmed, ">"); start >= 0 {
				start++
				if end := strings.LastIndex(trimmed, "</"); end >= 0 {
					snippet := trimHTML(trimmed[start:end])
					current.Content = strings.TrimSpace(snippet)
				}
			}
			if current.Title != "" {
				current.Title = trimHTML(current.Title)
				results = append(results, current)
				current = SearchResult{}
			}
			inResult = false
		} else if inResult && strings.Contains(trimmed, "</a>") {
			inResult = false
			if current.Title != "" {
				current.Title = trimHTML(current.Title)
				results = append(results, current)
				current = SearchResult{}
			}
		}

		if len(results) >= limit {
			break
		}
	}

	return results
}

func trimHTML(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "<b>", "")
	s = strings.ReplaceAll(s, "</b>", "")
	s = strings.ReplaceAll(s, "<em>", "")
	s = strings.ReplaceAll(s, "</em>", "")
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	return s
}
