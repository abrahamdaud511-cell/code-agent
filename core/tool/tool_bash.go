package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type BashTool struct{}

type BashArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
	Workdir string `json:"workdir,omitempty"`
}

func NewBashTool() *BashTool {
	return &BashTool{}
}

func (t *BashTool) Name() string {
	return "bash"
}

func (t *BashTool) Description() string {
	return "Execute shell commands in the project environment. Supports cross-platform execution."
}

func (t *BashTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Timeout in milliseconds (default: 120000)",
			},
			"workdir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory for the command",
			},
		},
		"required": []string{"command"},
	}
}

func (t *BashTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args BashArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	timeout := args.Timeout
	if timeout <= 0 {
		timeout = 120000
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" && !isPowerShell(args.Command) {
		cmd = exec.CommandContext(ctx, "cmd", "/c", args.Command)
	} else {
		cmd = exec.CommandContext(ctx, getShell(), getShellFlag(), args.Command)
	}

	if args.Workdir != "" {
		cmd.Dir = args.Workdir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	timer := time.AfterFunc(time.Duration(timeout)*time.Millisecond, func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	})
	defer timer.Stop()

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if err != nil {
		if output != "" {
			return output, fmt.Errorf("command failed: %w\nOutput: %s", err, output)
		}
		return "", fmt.Errorf("command failed: %w", err)
	}

	if len(output) > 50000 {
		output = output[:50000] + "\n... (truncated)"
	}

	return output, nil
}

func getShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	shell := "/bin/bash"
	if _, err := exec.LookPath("bash"); err != nil {
		shell = "/bin/sh"
	}
	return shell
}

func getShellFlag() string {
	if runtime.GOOS == "windows" {
		return "-Command"
	}
	return "-c"
}

func isPowerShell(cmd string) bool {
	return strings.Contains(cmd, "powershell") || strings.Contains(cmd, "pwsh")
}
