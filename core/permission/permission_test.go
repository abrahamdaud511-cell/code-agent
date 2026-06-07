package permission

import (
	"testing"

	"codeagent/config"
)

func TestNewManager(t *testing.T) {
	rules := []config.PermissionRule{
		{Tool: "bash", Action: "ask"},
		{Tool: "read", Action: "allow"},
	}
	m := NewManager(rules)

	if m == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestCheckExplicitRule(t *testing.T) {
	rules := []config.PermissionRule{
		{Tool: "bash", Action: "deny"},
		{Tool: "read", Action: "allow"},
	}
	m := NewManager(rules)

	action, err := m.Check("bash", "")
	if err != nil {
		t.Fatal(err)
	}
	if action != ActionDeny {
		t.Errorf("expected deny, got %s", action)
	}

	action, err = m.Check("read", "")
	if err != nil {
		t.Fatal(err)
	}
	if action != ActionAllow {
		t.Errorf("expected allow, got %s", action)
	}
}

func TestCheckDefaultRule(t *testing.T) {
	m := NewManager([]config.PermissionRule{})

	// bash defaults to ask
	action, err := m.Check("bash", "")
	if err != nil {
		t.Fatal(err)
	}
	if action != ActionAsk {
		t.Errorf("expected ask for bash, got %s", action)
	}

	// read defaults to allow
	action, err = m.Check("read", "")
	if err != nil {
		t.Fatal(err)
	}
	if action != ActionAllow {
		t.Errorf("expected allow for read, got %s", action)
	}
}

func TestCheckUnknownTool(t *testing.T) {
	m := NewManager([]config.PermissionRule{})

	action, err := m.Check("unknown-tool", "")
	if err != nil {
		t.Fatal(err)
	}
	if action != ActionAllow {
		t.Errorf("expected allow for unknown tool, got %s", action)
	}
}

func TestString(t *testing.T) {
	rules := []config.PermissionRule{
		{Tool: "bash", Action: "deny"},
		{Tool: "read", Action: "allow"},
	}
	m := NewManager(rules)

	str := m.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
	if !contains(str, "bash") {
		t.Error("expected string to contain bash")
	}
	if !contains(str, "read") {
		t.Error("expected string to contain read")
	}
}

func TestAddRule(t *testing.T) {
	m := NewManager([]config.PermissionRule{})
	m.AddRule("test-tool", ActionDeny)

	action, err := m.Check("test-tool", "")
	if err != nil {
		t.Fatal(err)
	}
	if action != ActionDeny {
		t.Errorf("expected deny, got %s", action)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
