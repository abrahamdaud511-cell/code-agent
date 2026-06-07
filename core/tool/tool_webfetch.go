package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type WebFetchTool struct{}
type WebFetchArgs struct {
	URL     string `json:"url"`
	Format  string `json:"format,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

var blockedHosts = []string{"169.254.169.254", "127.0.0.1", "localhost", "0.0.0.0", "::1", "metadata.google.internal", "metadata.internal", "100.100.100.200"}
var blockedCIDRs = []*net.IPNet{}

func init() {
	for _, cidr := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "169.254.0.0/16"} {
		if _, n, err := net.ParseCIDR(cidr); err == nil {
			blockedCIDRs = append(blockedCIDRs, n)
		}
	}
}

func isInternalHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	for _, b := range blockedHosts {
		if h == b || h == b+":" {
			return true
		}
	}
	ip := net.ParseIP(h)
	if ip == nil {
		if addrs, err := net.LookupHost(h); err == nil {
			for _, a := range addrs {
				if p := net.ParseIP(a); p != nil {
					for _, c := range blockedCIDRs {
						if c.Contains(p) {
							return true
						}
					}
				}
			}
		}
		return false
	}
	for _, c := range blockedCIDRs {
		if c.Contains(ip) {
			return true
		}
	}
	return false
}

func NewWebFetchTool() *WebFetchTool { return &WebFetchTool{} }
func (t *WebFetchTool) Name() string { return "webfetch" }
func (t *WebFetchTool) Description() string { return "Fetch web content from a public URL." }
func (t *WebFetchTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url":     map[string]interface{}{"type": "string", "description": "The URL to fetch"},
			"format":  map[string]interface{}{"type": "string", "enum": []string{"markdown", "text", "html"}},
			"timeout": map[string]interface{}{"type": "integer", "description": "Timeout in seconds"},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args WebFetchArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.URL == "" {
		return "", fmt.Errorf("url is required")
	}
	if !strings.HasPrefix(args.URL, "https://") {
		return "", fmt.Errorf("only HTTPS URLs are allowed")
	}
	parsed, err := url.Parse(args.URL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if isInternalHost(parsed.Hostname()) {
		return "", fmt.Errorf("access denied: %s is restricted", parsed.Hostname())
	}
	timeout := args.Timeout
	if timeout <= 0 { timeout = 30 }
	if timeout > 60 { timeout = 60 }
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			if isInternalHost(req.URL.Hostname()) {
				return fmt.Errorf("redirect to internal host blocked")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, "GET", args.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "CodeAgent/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), nil
	}
	content := string(body)
	if len(content) > 50000 {
		content = content[:50000] + "\n... (truncated)"
	}
	return fmt.Sprintf("URL: %s\nStatus: %d\nContent-Type: %s\n\n%s", args.URL, resp.StatusCode, resp.Header.Get("Content-Type"), content), nil
}
