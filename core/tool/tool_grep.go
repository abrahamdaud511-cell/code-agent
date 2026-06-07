package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type GrepTool struct{}

type GrepArgs struct {
	Pattern string `json:"pattern"`
	Include string `json:"include,omitempty"`
	Path    string `json:"path,omitempty"`
	IgnoreCase bool `json:"ignoreCase,omitempty"`
}

func NewGrepTool() *GrepTool {
	return &GrepTool{}
}

func (t *GrepTool) Name() string {
	return "grep"
}

func (t *GrepTool) Description() string {
	return "Search file contents using regular expressions. Fast content search across the codebase."
}

func (t *GrepTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The regex pattern to search for",
			},
			"include": map[string]interface{}{
				"type":        "string",
				"description": "File pattern to include (e.g., *.go, *.{ts,tsx})",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory to search in (default: current)",
			},
			"ignoreCase": map[string]interface{}{
				"type":        "boolean",
				"description": "Case insensitive search",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args GrepArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	// Use ripgrep if available, fallback to grep
	var cmd *exec.Cmd

	if _, err := exec.LookPath("rg"); err == nil {
		rgArgs := []string{"--line-number", "--with-filename", "--color", "never"}
		if args.IgnoreCase {
			rgArgs = append(rgArgs, "-i")
		}
		if args.Include != "" {
			rgArgs = append(rgArgs, "-g", args.Include)
		}
		if args.Path != "" {
			rgArgs = append(rgArgs, args.Path)
		}
		rgArgs = append(rgArgs, args.Pattern)
		cmd = exec.CommandContext(ctx, "rg", rgArgs...)
	} else if runtime.GOOS == "windows" {
		// Use findstr on Windows
		cmd = exec.CommandContext(ctx, "findstr", "/s", "/n", args.Pattern)
		if args.Path != "" {
			cmd.Dir = args.Path
		}
	} else {
		// Fallback to grep
		grepArgs := []string{"-rn", "--color=never"}
		if args.IgnoreCase {
			grepArgs = append(grepArgs, "-i")
		}
		if args.Include != "" {
			grepArgs = append(grepArgs, "--include="+args.Include)
		}
		if args.Path != "" {
			grepArgs = append(grepArgs, args.Path)
		} else {
			grepArgs = append(grepArgs, ".")
		}
		grepArgs = append(grepArgs, "-e", args.Pattern)
		cmd = exec.CommandContext(ctx, "grep", grepArgs...)
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "No matches found", nil
		}
		return "", fmt.Errorf("search failed: %w", err)
	}

	result := string(output)
	if len(result) > 10000 {
		lines := strings.Split(result, "\n")
		result = strings.Join(lines[:200], "\n")
		result += fmt.Sprintf("\n... and %d more matches", len(lines)-200)
	}

	return result, nil
}
