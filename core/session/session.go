package session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"codeagent/config"
)

type Session struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Model      string    `json:"model"`
	AgentName  string    `json:"agent_name"`
	ProjectDir string    `json:"project_dir"`

	Messages    []Message        `json:"messages"`
	Attachments []string         `json:"attachments"`
	Metadata    map[string]string `json:"metadata"`

	mu        sync.RWMutex
	store     *Store
	compacted bool
	gitCommit string
}

type Message struct {
	Role        string       `json:"role"`
	Content     string       `json:"content"`
	ToolCalls   []ToolCall   `json:"tool_calls,omitempty"`
	ToolResults []ToolResult `json:"tool_results,omitempty"`
	Timestamp   time.Time    `json:"timestamp"`
}

type ToolCall struct {
	ID     string `json:"id"`
	Tool   string `json:"tool"`
	Input  string `json:"input"`
	Status string `json:"status"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	Error      string `json:"error,omitempty"`
}

func New(cfg *config.Config, store *Store) *Session {
	dir, _ := os.Getwd()
	s := &Session{
		ID:         generateID(),
		Title:      "New Session",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Model:      cfg.DefaultModel,
		AgentName:  "build",
		ProjectDir: dir,
		Messages:   make([]Message, 0),
		store:      store,
	}
	if s.Model == "" {
		s.Model = "openai/gpt-5"
	}
	return s
}

func (s *Session) AddMessage(role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, Message{
		Role: role, Content: content, Timestamp: time.Now(),
	})
	s.UpdatedAt = time.Now()
	if role == "user" && s.Title == "New Session" && len(content) > 0 {
		if len(content) > 60 {
			s.Title = content[:60] + "..."
		} else {
			s.Title = content
		}
	}
	s.trySave()
}

func (s *Session) AddToolCall(tc ToolCall) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Messages) > 0 {
		last := &s.Messages[len(s.Messages)-1]
		if last.Role == "assistant" {
			last.ToolCalls = append(last.ToolCalls, tc)
		}
	}
	s.UpdatedAt = time.Now()
}

func (s *Session) AddToolResult(toolCallID, content string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tr := ToolResult{ToolCallID: toolCallID, Content: content}
	if err != nil {
		tr.Error = err.Error()
	}
	s.Messages = append(s.Messages, Message{
		Role: "tool", Content: content, ToolResults: []ToolResult{tr}, Timestamp: time.Now(),
	})
	s.UpdatedAt = time.Now()
	s.trySave()
}

func (s *Session) AttachFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Attachments = append(s.Attachments, path)
}

func (s *Session) SetModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Model = model
}

func (s *Session) SetAgent(agent string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AgentName = agent
}

func (s *Session) GetHistory() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := make([]Message, len(s.Messages))
	copy(msgs, s.Messages)
	return msgs
}

func (s *Session) Fork() *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := make([]Message, len(s.Messages))
	copy(msgs, s.Messages)
	return &Session{
		ID: generateID(), Title: "Fork: " + s.Title,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
		Model: s.Model, AgentName: s.AgentName, ProjectDir: s.ProjectDir,
		Messages: msgs, store: s.store,
	}
}

func (s *Session) Undo() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Messages) < 2 {
		return fmt.Errorf("nothing to undo")
	}
	if isGitRepo(s.ProjectDir) {
		s.gitCommit = execGitOutput(s.ProjectDir, "rev-parse", "HEAD")
		execGit(s.ProjectDir, "checkout", ".")
	}
	s.Messages = s.Messages[:len(s.Messages)-2]
	s.UpdatedAt = time.Now()
	s.trySave()
	return nil
}

func (s *Session) Redo() error {
	if s.gitCommit == "" {
		return fmt.Errorf("nothing to redo")
	}
	if isGitRepo(s.ProjectDir) {
		execGit(s.ProjectDir, "checkout", s.gitCommit)
	}
	s.gitCommit = ""
	return nil
}

func (s *Session) Compact() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.compacted || len(s.Messages) <= 4 {
		return nil
	}
	keep := []Message{s.Messages[0]}
	if len(s.Messages) > 4 {
		keep = append(keep, s.Messages[len(s.Messages)-4:]...)
	} else {
		keep = append(keep, s.Messages[1:]...)
	}
	s.Messages = keep
	s.compacted = true
	s.UpdatedAt = time.Now()
	s.trySave()
	return nil
}

func (s *Session) Share() (string, error) {
	return fmt.Sprintf("https://codeagent.ai/share/%s", s.ID), nil
}

func (s *Session) Export() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# CodeAgent Session: %s\n\n", s.Title))
	b.WriteString(fmt.Sprintf("- Date: %s\n", s.CreatedAt.Format("2006-01-02 15:04")))
	b.WriteString(fmt.Sprintf("- Model: %s\n", s.Model))
	b.WriteString(fmt.Sprintf("- Messages: %d\n\n", len(s.Messages)))
	b.WriteString("---\n\n")
	for _, msg := range s.Messages {
		role := "## " + msg.Role
		if msg.Role == "user" {
			role = "## You"
		} else if msg.Role == "assistant" {
			role = "## CodeAgent"
		}
		b.WriteString(fmt.Sprintf("%s\n\n%s\n\n", role, msg.Content))
		for _, tc := range msg.ToolCalls {
			b.WriteString(fmt.Sprintf("> 🔧 %s\n> \n> ```\n> %s\n> ```\n\n", tc.Tool, tc.Input))
		}
	}
	return b.String(), nil
}

func (s *Session) GetContext() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.Messages) == 0 {
		return ""
	}
	return s.Messages[len(s.Messages)-1].Content
}

func (s *Session) trySave() {
	if s.store != nil {
		data, _ := json.Marshal(s)
		go s.store.SaveRaw(s.ID, string(data))
	}
}

func generateID() string {
	b := make([]byte, 16)
	n := time.Now().UnixNano()
	for i := 0; i < 8; i++ {
		b[i] = byte(n >> (i * 8))
	}
	return fmt.Sprintf("%x", b)
}

func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func execGit(dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Run()
}

func execGitOutput(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}
