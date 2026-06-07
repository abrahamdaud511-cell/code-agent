package session

import (
	"os"
	"path/filepath"
	"testing"

	"codeagent/config"
)

func TestNewSession(t *testing.T) {
	cfg := config.DefaultConfig()
	store, err := NewStore(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sess := New(cfg, store)
	if sess == nil {
		t.Fatal("expected non-nil session")
	}
	if sess.ID == "" {
		t.Error("expected non-empty ID")
	}
	if sess.Title != "New Session" {
		t.Errorf("expected 'New Session', got %s", sess.Title)
	}
	if sess.AgentName != "build" {
		t.Errorf("expected 'build' agent, got %s", sess.AgentName)
	}
}

func TestAddMessage(t *testing.T) {
	cfg := config.DefaultConfig()
	store, err := NewStore(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sess := New(cfg, store)
	sess.AddMessage("user", "Hello")

	if len(sess.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(sess.Messages))
	}
	if sess.Messages[0].Role != "user" {
		t.Errorf("expected role 'user', got %s", sess.Messages[0].Role)
	}
	if sess.Messages[0].Content != "Hello" {
		t.Errorf("expected content 'Hello', got %s", sess.Messages[0].Content)
	}
}

func TestGetHistory(t *testing.T) {
	cfg := config.DefaultConfig()
	store, err := NewStore(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sess := New(cfg, store)
	sess.AddMessage("user", "Hi")
	sess.AddMessage("assistant", "Hello!")

	history := sess.GetHistory()
	if len(history) != 2 {
		t.Errorf("expected 2 messages, got %d", len(history))
	}
}

func TestFork(t *testing.T) {
	cfg := config.DefaultConfig()
	store, err := NewStore(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sess := New(cfg, store)
	sess.AddMessage("user", "Original message")

	forked := sess.Fork()
	if forked.ID == sess.ID {
		t.Error("forked session should have different ID")
	}
	if !contains(forked.Title, "Fork") {
		t.Errorf("expected forked title to contain 'Fork', got %s", forked.Title)
	}
	if len(forked.Messages) != len(sess.Messages) {
		t.Errorf("expected same message count, got %d vs %d", len(forked.Messages), len(sess.Messages))
	}
}

func TestExport(t *testing.T) {
	cfg := config.DefaultConfig()
	store, err := NewStore(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sess := New(cfg, store)
	sess.AddMessage("user", "Test message")

	export, err := sess.Export()
	if err != nil {
		t.Fatal(err)
	}
	if export == "" {
		t.Error("expected non-empty export")
	}
	if !contains(export, "Test message") {
		t.Error("expected export to contain message content")
	}
}

func TestStore(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	cfg := config.DefaultConfig()
	sess := New(cfg, store)
	sess.AddMessage("user", "Save test")

	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Model != sess.Model {
		t.Errorf("expected model %s, got %s", sess.Model, loaded.Model)
	}
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	cfg := config.DefaultConfig()

	sess1 := New(cfg, store)
	sess1.AddMessage("user", "Session 1")
	store.Save(sess1)

	sess2 := New(cfg, store)
	sess2.AddMessage("user", "Session 2")
	store.Save(sess2)

	sessions, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) < 2 {
		t.Errorf("expected at least 2 sessions, got %d", len(sessions))
	}
}

func TestStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	cfg := config.DefaultConfig()
	sess := New(cfg, store)
	store.Save(sess)

	if err := store.Delete(sess.ID); err != nil {
		t.Fatal(err)
	}

	_, err = store.Load(sess.ID)
	if err == nil {
		t.Error("expected error loading deleted session")
	}
}

func TestSetModel(t *testing.T) {
	cfg := config.DefaultConfig()
	sess := New(cfg, nil)
	sess.SetModel("test/model")

	if sess.Model != "test/model" {
		t.Errorf("expected 'test/model', got %s", sess.Model)
	}
}

func TestSetAgent(t *testing.T) {
	cfg := config.DefaultConfig()
	sess := New(cfg, nil)
	sess.SetAgent("plan")

	if sess.AgentName != "plan" {
		t.Errorf("expected 'plan', got %s", sess.AgentName)
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

func TestMain(m *testing.M) {
	// Use temp dir for test data
	tmpDir, _ := os.MkdirTemp("", "codeagent-test-*")
	os.Setenv("CODEAGENT_DATA_DIR", filepath.Join(tmpDir, "data"))
	defer os.RemoveAll(tmpDir)

	code := m.Run()
	os.Exit(code)
}
