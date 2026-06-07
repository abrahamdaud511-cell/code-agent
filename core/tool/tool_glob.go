package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"io/fs"
)

type GlobTool struct{}

type GlobArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

func NewGlobTool() *GlobTool {
	return &GlobTool{}
}

func (t *GlobTool) Name() string {
	return "glob"
}

func (t *GlobTool) Description() string {
	return "Find files by pattern matching. Supports glob patterns like **/*.js or src/**/*.ts."
}

func (t *GlobTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The glob pattern to match files against",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The directory to search in",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GlobTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args GlobArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	searchDir := args.Path
	if searchDir == "" {
		searchDir, _ = os.Getwd()
	}

	matches, err := filepath.Glob(filepath.Join(searchDir, args.Pattern))
	if err != nil {
		return "", fmt.Errorf("invalid glob pattern: %w", err)
	}

	// If no matches with basic glob, try recursive
	if len(matches) == 0 && strings.Contains(args.Pattern, "**") {
		matches = t.walkGlob(searchDir, args.Pattern)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i] < matches[j]
	})

	if len(matches) > 100 {
		matches = matches[:100]
	}

	if len(matches) == 0 {
		return "No files found matching pattern", nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d file(s) matching '%s':\n", len(matches), args.Pattern))
	for _, m := range matches {
		result.WriteString(m)
		result.WriteByte('\n')
	}

	return result.String(), nil
}

func (t *GlobTool) walkGlob(root, pattern string) []string {
	var matches []string
	parts := strings.SplitN(pattern, "**", 2)

	if len(parts) != 2 {
		return matches
	}

	_ = parts[0]
	suffix := parts[1]

	filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(path, suffix) {
			matches = append(matches, path)
		}
		return nil
	})

	return matches
}
