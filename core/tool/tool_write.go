package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type WriteTool struct{}

type WriteArgs struct {
	FilePath string `json:"filePath"`
	Content  string `json:"content"`
}

func NewWriteTool() *WriteTool {
	return &WriteTool{}
}

func (t *WriteTool) Name() string {
	return "write"
}

func (t *WriteTool) Description() string {
	return "Create new files or overwrite existing ones."
}

func (t *WriteTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"filePath": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to write",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"filePath", "content"},
	}
}

func (t *WriteTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args WriteArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.FilePath == "" {
		return "", fmt.Errorf("filePath is required")
	}

	dir := filepath.Dir(args.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directories: %w", err)
	}

	if err := os.WriteFile(args.FilePath, []byte(args.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	info, err := os.Stat(args.FilePath)
	if err != nil {
		return fmt.Sprintf("Created file: %s", args.FilePath), nil
	}

	return fmt.Sprintf("Written %d bytes to %s", info.Size(), args.FilePath), nil
}
