package tool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type QuestionTool struct {
	questionChan chan string
	answerChan   chan string
	interactive  bool
}

type QuestionArgs struct {
	Question string   `json:"question"`
	Header   string   `json:"header,omitempty"`
	Options  []Option `json:"options,omitempty"`
	Multiple bool     `json:"multiple,omitempty"`
	Default  string   `json:"default,omitempty"`
}

type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

func NewQuestionTool() *QuestionTool {
	return &QuestionTool{
		questionChan: make(chan string, 10),
		answerChan:   make(chan string, 10),
		interactive:  false,
	}
}

func (t *QuestionTool) Name() string {
	return "question"
}

func (t *QuestionTool) Description() string {
	return "Ask the user questions during execution. Supports multiple choice, text input, and confirmation prompts. Use this to gather preferences, clarify intent, or get approval."
}

func (t *QuestionTool) Parameters() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"question": map[string]interface{}{
				"type":        "string",
				"description": "The question to ask the user",
			},
			"header": map[string]interface{}{
				"type":        "string",
				"description": "Short label for the question category",
			},
			"options": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"label": map[string]interface{}{
							"type":        "string",
							"description": "Display text for the option",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Explanation of the option",
						},
					},
					"required": []string{"label"},
				},
				"description": "Available choices for the user",
			},
			"multiple": map[string]interface{}{
				"type":        "boolean",
				"description": "Allow selecting multiple options",
			},
			"default": map[string]interface{}{
				"type":        "string",
				"description": "Default answer if user presses enter without typing",
			},
		},
		"required": []string{"question"},
	}
}

const (
	questionInteractivePrompt = "[question] "
	questionResponsePrefix    = "[response] "
)

func (t *QuestionTool) Execute(ctx context.Context, argsJson json.RawMessage) (string, error) {
	var args QuestionArgs
	if err := json.Unmarshal(argsJson, &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Question == "" {
		return "", fmt.Errorf("question is required")
	}

	if len(args.Options) > 0 {
		return t.askMultipleChoice(ctx, args)
	}

	return t.askTextInput(ctx, args)
}

func (t *QuestionTool) askMultipleChoice(ctx context.Context, args QuestionArgs) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n=== %s ===\n\n", args.Header))
	sb.WriteString(fmt.Sprintf("Question: %s\n\n", args.Question))

	sb.WriteString("Options:\n")
	for i, opt := range args.Options {
		sb.WriteString(fmt.Sprintf("  %d. %s", i+1, opt.Label))
		if opt.Description != "" {
			sb.WriteString(fmt.Sprintf(" - %s", opt.Description))
		}
		sb.WriteString("\n")
	}

	if args.Multiple {
		sb.WriteString("\nSelect options (e.g., \"1,3,5\"): ")
	} else {
		sb.WriteString(fmt.Sprintf("\nSelect option (1-%d): ", len(args.Options)))
	}

	fmt.Fprint(os.Stderr, sb.String())

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	input = strings.TrimSpace(input)

	if input == "" && args.Default != "" {
		input = args.Default
	}

	if args.Multiple {
		parts := strings.Split(input, ",")
		var selected []string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if idx, err := strconv.Atoi(part); err == nil && idx >= 1 && idx <= len(args.Options) {
				option := args.Options[idx-1]
				selected = append(selected, option.Label)
			}
		}
		if len(selected) == 0 {
			return "", fmt.Errorf("invalid selection. Please enter numbers between 1 and %d", len(args.Options))
		}
		result := fmt.Sprintf("Selected: %s", strings.Join(selected, ", "))
		fmt.Fprintf(os.Stderr, "%s%s\n", questionResponsePrefix, result)
		return result, nil
	}

	if idx, err := strconv.Atoi(input); err == nil && idx >= 1 && idx <= len(args.Options) {
		option := args.Options[idx-1]
		result := fmt.Sprintf("Selected: %s", option.Label)
		if option.Description != "" {
			result += fmt.Sprintf(" - %s", option.Description)
		}
		fmt.Fprintf(os.Stderr, "%s%s\n", questionResponsePrefix, result)
		return result, nil
	}

	// Try matching by label
	for _, opt := range args.Options {
		if strings.EqualFold(input, opt.Label) {
			result := fmt.Sprintf("Selected: %s", opt.Label)
			if opt.Description != "" {
				result += fmt.Sprintf(" - %s", opt.Description)
			}
			fmt.Fprintf(os.Stderr, "%s%s\n", questionResponsePrefix, result)
			return result, nil
		}
	}

	return "", fmt.Errorf("invalid selection: %s. Please choose a number between 1 and %d", input, len(args.Options))
}

func (t *QuestionTool) askTextInput(ctx context.Context, args QuestionArgs) (string, error) {
	var sb strings.Builder

	if args.Header != "" {
		sb.WriteString(fmt.Sprintf("=== %s ===\n", args.Header))
	}
	sb.WriteString(fmt.Sprintf("%s: ", args.Question))
	if args.Default != "" {
		sb.WriteString(fmt.Sprintf(" (default: %s)", args.Default))
	}

	fmt.Fprint(os.Stderr, sb.String())

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	input = strings.TrimSpace(input)

	if input == "" && args.Default != "" {
		input = args.Default
	}

	if input == "" {
		return "", fmt.Errorf("input is required")
	}

	result := fmt.Sprintf("User response: %s", input)
	fmt.Fprintf(os.Stderr, "%s%s\n", questionResponsePrefix, result)

	time.Sleep(50 * time.Millisecond)
	return result, nil
}

func (t *QuestionTool) SetInteractive(interactive bool) {
	t.interactive = interactive
}
