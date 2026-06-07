package permission

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codeagent/config"
)

type Action string

const (
	ActionAllow Action = "allow"
	ActionDeny  Action = "deny"
	ActionAsk   Action = "ask"
)

type Rule struct {
	Tool   string `json:"tool"`
	Action Action `json:"action"`
	Path   string `json:"path,omitempty"`
}

type Manager struct {
	rules    []Rule
	defaults map[string]Action
}

func NewManager(permRules []config.PermissionRule) *Manager {
	m := &Manager{
		rules:    make([]Rule, 0),
		defaults: make(map[string]Action),
	}

	for _, rule := range permRules {
		m.rules = append(m.rules, Rule{
			Tool:   rule.Tool,
			Action: Action(rule.Action),
		})
	}

	m.defaults["bash"] = ActionAsk
	m.defaults["read"] = ActionAllow
	m.defaults["edit"] = ActionAsk
	m.defaults["write"] = ActionAsk
	m.defaults["grep"] = ActionAllow
	m.defaults["glob"] = ActionAllow
	m.defaults["webfetch"] = ActionAllow
	m.defaults["websearch"] = ActionAsk
	m.defaults["question"] = ActionAllow
	m.defaults["task"] = ActionAsk
	m.defaults["todowrite"] = ActionAsk
	m.defaults["mouse_move"] = ActionAsk
	m.defaults["mouse_click"] = ActionAsk
	m.defaults["mouse_scroll"] = ActionAsk
	m.defaults["keyboard_type"] = ActionAsk
	m.defaults["keyboard_press"] = ActionAsk

	return m
}

func (m *Manager) Check(tool, path string) (Action, error) {
	for _, rule := range m.rules {
		if matched, _ := filepath.Match(rule.Tool, tool); matched {
			return rule.Action, nil
		}
		if strings.Contains(rule.Tool, "*") {
			if matched, _ := filepath.Match(rule.Tool, tool); matched {
				return rule.Action, nil
			}
		}
		if rule.Tool == tool {
			return rule.Action, nil
		}
	}

	if action, ok := m.defaults[tool]; ok {
		return action, nil
	}

	return ActionAllow, nil
}

func (m *Manager) Prompt(tool, input string) bool {
	fmt.Fprintf(os.Stderr, "\n⚠️  Tool: %s\n", bold(tool))
	fmt.Fprintf(os.Stderr, "   Input: %s\n", truncateStr(input, 200))
	fmt.Fprintf(os.Stderr, "   Allow this operation? [y]es / [n]o / [a]llow always / [d]eny always: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	switch response {
	case "y", "yes", "":
		return true
	case "a", "allow", "always":
		m.rules = append(m.rules, Rule{Tool: tool, Action: ActionAllow})
		fmt.Fprintf(os.Stderr, "   ✓ Rule saved: %s → allow\n", tool)
		return true
	case "d", "deny", "never":
		m.rules = append(m.rules, Rule{Tool: tool, Action: ActionDeny})
		fmt.Fprintf(os.Stderr, "   ✗ Rule saved: %s → deny\n", tool)
		return false
	default:
		return false
	}
}

func (m *Manager) String() string {
	var result strings.Builder
	result.WriteString("Permission Rules:\n")

	allTools := []string{"bash", "read", "write", "edit", "grep", "glob", "webfetch", "websearch", "question", "task", "todowrite", "mouse_move", "mouse_click", "mouse_scroll", "keyboard_type", "keyboard_press"}
	ruleMap := make(map[string]string)
	for _, rule := range m.rules {
		ruleMap[rule.Tool] = string(rule.Action)
	}

	for _, tool := range allTools {
		action := ruleMap[tool]
		if action == "" {
			if a, ok := m.defaults[tool]; ok {
				action = string(a)
			} else {
				action = "allow"
			}
		}

		status := "✓"
		if action == "deny" {
			status = "✗"
		} else if action == "ask" {
			status = "?"
		}

		result.WriteString(fmt.Sprintf("  %s %s: %s\n", status, tool, action))
	}

	return result.String()
}

func (m *Manager) AddRule(tool string, action Action) {
	m.rules = append(m.rules, Rule{Tool: tool, Action: action})
}

func truncateStr(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func bold(s string) string {
	return "\033[1m" + s + "\033[0m"
}
