package tool

import (
	"context"
	"encoding/json"
	"testing"

	"codeagent/config"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry([]config.PermissionRule{})
	if r == nil {
		t.Fatal("expected non-nil registry")
	}

	tools := r.List()
	expectedTools := []string{"bash", "read", "write", "edit", "grep", "glob", "webfetch", "websearch", "question", "task", "todowrite"}

	for _, name := range expectedTools {
		found := false
		for _, tName := range tools {
			if tName == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tool %s to be registered", name)
		}
	}
}

func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry([]config.PermissionRule{})

	tool, ok := r.Get("read")
	if !ok {
		t.Fatal("expected read tool to be found")
	}
	if tool.Name() != "read" {
		t.Errorf("expected name 'read', got %s", tool.Name())
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent tool to not be found")
	}
}

func TestDefinitions(t *testing.T) {
	r := NewRegistry([]config.PermissionRule{})
	defs := r.Definitions()

	if len(defs) == 0 {
		t.Error("expected non-empty definitions")
	}

	names := make(map[string]bool)
	for _, d := range defs {
		if names[d.Function.Name] {
			t.Errorf("duplicate definition: %s", d.Function.Name)
		}
		names[d.Function.Name] = true

		if d.Function.Description == "" {
			t.Errorf("missing description for %s", d.Function.Name)
		}
		if d.Function.Parameters == nil {
			t.Errorf("missing parameters for %s", d.Function.Name)
		}
	}
}

func TestExecute(t *testing.T) {
	r := NewRegistry([]config.PermissionRule{})

	args, _ := json.Marshal(map[string]interface{}{
		"filePath": "test.txt",
		"content":  "hello",
	})

	// Write tool
	result, err := r.Execute(context.Background(), "write", string(args))
	if err != nil {
		t.Errorf("write failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestExecuteDeniedTool(t *testing.T) {
	rules := []config.PermissionRule{
		{Tool: "bash", Action: "deny"},
	}
	r := NewRegistry(rules)

	_, err := r.Execute(context.Background(), "bash", `{"command": "echo hello"}`)
	if err == nil {
		t.Error("expected error for denied tool")
	}
}

func TestExecuteUnknownTool(t *testing.T) {
	r := NewRegistry([]config.PermissionRule{})

	_, err := r.Execute(context.Background(), "nonexistent", "{}")
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestBashTool(t *testing.T) {
	tool := NewBashTool()
	if tool.Name() != "bash" {
		t.Errorf("expected name 'bash', got %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestReadTool(t *testing.T) {
	tool := NewReadTool()
	if tool.Name() != "read" {
		t.Errorf("expected name 'read', got %s", tool.Name())
	}
}

func TestWriteTool(t *testing.T) {
	tool := NewWriteTool()
	if tool.Name() != "write" {
		t.Errorf("expected name 'write', got %s", tool.Name())
	}
}

func TestEditTool(t *testing.T) {
	tool := NewEditTool()
	if tool.Name() != "edit" {
		t.Errorf("expected name 'edit', got %s", tool.Name())
	}
}

func TestGrepTool(t *testing.T) {
	tool := NewGrepTool()
	if tool.Name() != "grep" {
		t.Errorf("expected name 'grep', got %s", tool.Name())
	}
}

func TestGlobTool(t *testing.T) {
	tool := NewGlobTool()
	if tool.Name() != "glob" {
		t.Errorf("expected name 'glob', got %s", tool.Name())
	}
}

func TestWebFetchTool(t *testing.T) {
	tool := NewWebFetchTool()
	if tool.Name() != "webfetch" {
		t.Errorf("expected name 'webfetch', got %s", tool.Name())
	}
}

func TestWebSearchTool(t *testing.T) {
	tool := NewWebSearchTool()
	if tool.Name() != "websearch" {
		t.Errorf("expected name 'websearch', got %s", tool.Name())
	}
}

func TestQuestionTool(t *testing.T) {
	tool := NewQuestionTool()
	if tool.Name() != "question" {
		t.Errorf("expected name 'question', got %s", tool.Name())
	}
}

func TestTaskTool(t *testing.T) {
	tool := NewTaskTool()
	if tool.Name() != "task" {
		t.Errorf("expected name 'task', got %s", tool.Name())
	}
}

func TestTodoWriteTool(t *testing.T) {
	tool := NewTodoWriteTool()
	if tool.Name() != "todowrite" {
		t.Errorf("expected name 'todowrite', got %s", tool.Name())
	}
}
