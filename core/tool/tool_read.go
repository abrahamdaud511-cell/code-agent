package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadTool struct{}
type ReadArgs struct {
	FilePath string `json:"filePath"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

func NewReadTool() *ReadTool { return &ReadTool{} }
func (t *ReadTool) Name() string { return "read" }
func (t *ReadTool) Description() string { return "Read file contents from the project." }
func (t *ReadTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"filePath": map[string]interface{}{"type": "string", "description": "The absolute path to the file to read"},
			"offset":   map[string]interface{}{"type": "integer", "description": "Line number to start from (1-indexed)"},
			"limit":    map[string]interface{}{"type": "integer", "description": "Maximum number of lines"},
		},
		"required": []string{"filePath"},
	}
}

var blockedPaths = []string{"/etc/", "/proc/", "/sys/", "/dev/", "/boot/", "/var/log", "/var/db", "~/.ssh", "/root/.ssh", "/home/", "/Users/", ".git/", "go.mod", "go.sum"}

func isBlockedPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil { return fmt.Errorf("invalid path: %w", err) }
	lower := strings.ToLower(abs)
	for _, blocked := range blockedPaths {
		ba, _ := filepath.Abs(blocked)
		if ba != "" && strings.HasPrefix(lower, strings.ToLower(ba)) {
			return fmt.Errorf("access denied: %s is restricted", path)
		}
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("access denied: path traversal not allowed")
	}
	return nil
}

func (t *ReadTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args ReadArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.FilePath == "" {
		return "", fmt.Errorf("filePath is required")
	}
	if err := isBlockedPath(args.FilePath); err != nil {
		return "", err
	}
	data, err := os.ReadFile(args.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", args.FilePath)
		}
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	content := string(data)
	lines := strings.Split(content, "\n")
	if args.Offset > 0 || args.Limit > 0 {
		start := args.Offset
		if start <= 0 { start = 1 }
		end := len(lines)
		if args.Limit > 0 {
			end = start + args.Limit - 1
			if end > len(lines) { end = len(lines) }
		}
		if start > len(lines) {
			return "", fmt.Errorf("offset %d exceeds file length %d", start, len(lines))
		}
		lines = lines[start-1 : end]
	}
	var result strings.Builder
	for i, line := range lines {
		n := i + 1
		if args.Offset > 0 { n = args.Offset + i }
		if len(line) > 2000 { line = line[:2000] + "..." }
		result.WriteString(fmt.Sprintf("%d: %s\n", n, line))
	}
	return result.String(), nil
}
