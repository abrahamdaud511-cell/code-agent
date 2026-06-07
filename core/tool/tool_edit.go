package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type EditTool struct{}

type EditArgs struct {
	FilePath  string `json:"filePath"`
	OldString string `json:"oldString"`
	NewString string `json:"newString"`
	ReplaceAll bool   `json:"replaceAll,omitempty"`
}

func NewEditTool() *EditTool {
	return &EditTool{}
}

func (t *EditTool) Name() string {
	return "edit"
}

func (t *EditTool) Description() string {
	return "Modify existing files using exact string replacements."
}

func (t *EditTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"filePath": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"oldString": map[string]interface{}{
				"type":        "string",
				"description": "The text to replace",
			},
			"newString": map[string]interface{}{
				"type":        "string",
				"description": "The text to replace it with",
			},
			"replaceAll": map[string]interface{}{
				"type":        "boolean",
				"description": "Replace all occurrences of oldString",
			},
		},
		"required": []string{"filePath", "oldString", "newString"},
	}
}

func (t *EditTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args EditArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.FilePath == "" {
		return "", fmt.Errorf("filePath is required")
	}

	if args.OldString == "" {
		return "", fmt.Errorf("oldString is required")
	}

	data, err := os.ReadFile(args.FilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	// Try multiple matching strategies
	replacements := 0
	if args.ReplaceAll {
		newContent := strings.ReplaceAll(content, args.OldString, args.NewString)
		replacements = strings.Count(content, args.OldString)
		content = newContent
	} else {
		if strings.Contains(content, args.OldString) {
			content = strings.Replace(content, args.OldString, args.NewString, 1)
			replacements = 1
		} else {
			// Try with normalized whitespace
			normalizedOld := normalizeWS(args.OldString)
			normalizedContent := normalizeWS(content)
			if idx := strings.Index(normalizedContent, normalizedOld); idx >= 0 {
				// Find the exact original text
				originalStart := findOriginalAtIndex(content, normalizedContent, idx)
				if originalStart >= 0 {
					content = content[:originalStart] + args.NewString + content[originalStart+len(args.OldString):]
					replacements = 1
				}
			}
		}
	}

	if replacements == 0 {
		return "", fmt.Errorf("oldString not found in file")
	}

	if err := os.WriteFile(args.FilePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Applied edit to %s (%d replacement(s))", args.FilePath, replacements), nil
}

func normalizeWS(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

func findOriginalAtIndex(content, normalizedContent string, idx int) int {
	if idx >= len(normalizedContent) {
		return -1
	}

	// Find corresponding position in original content
	origIdx := 0
	normIdx := 0
	for normIdx < idx && origIdx < len(content) {
		if content[origIdx] == '\n' || content[origIdx] == ' ' || content[origIdx] == '\t' {
			origIdx++
			continue
		}
		if normalizedContent[normIdx] != ' ' && normalizedContent[normIdx] != '\n' && normalizedContent[normIdx] != '\t' {
			normIdx++
		}
		if normIdx < idx {
			origIdx++
		}
	}
	return origIdx
}
