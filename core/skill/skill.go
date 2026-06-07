package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	FilePath    string `json:"file_path"`
	MatchPattern string `json:"match_pattern,omitempty"`
}

type Manager struct {
	mu       sync.RWMutex
	skills   map[string]*Skill
	skillDir string
}

func NewManager(skillDir string) *Manager {
	return &Manager{
		skills:   make(map[string]*Skill),
		skillDir: skillDir,
	}
}

func (m *Manager) LoadAll() error {
	if m.skillDir == "" {
		return nil
	}

	if _, err := os.Stat(m.skillDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(m.skillDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".txt") && !strings.HasSuffix(name, ".json") {
			continue
		}

		path := filepath.Join(m.skillDir, name)
		if err := m.Load(name, path); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load skill %s: %v\n", name, err)
		}
	}

	return nil
}

func (m *Manager) Load(name, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read skill file: %w", err)
	}

	skillName := strings.TrimSuffix(name, filepath.Ext(name))
	description := fmt.Sprintf("Skill from %s", path)

	content := string(data)

	// Parse frontmatter if present (YAML-style --- blocks)
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content[3:], "---", 2)
		if len(parts) == 2 {
			frontmatter := strings.TrimSpace(parts[0])
			content = strings.TrimSpace(parts[1])

			for _, line := range strings.Split(frontmatter, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "name:") {
					skillName = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				} else if strings.HasPrefix(line, "description:") {
					description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				} else if strings.HasPrefix(line, "match:") {
					// match pattern handled here
				}
			}
		}
	}

	m.skills[skillName] = &Skill{
		Name:        skillName,
		Description: description,
		Content:     content,
		FilePath:    path,
	}

	return nil
}

func (m *Manager) Get(name string) (*Skill, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.skills[name]
	return s, ok
}

func (m *Manager) List() []*Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Skill, 0, len(m.skills))
	for _, s := range m.skills {
		list = append(list, s)
	}
	return list
}

func (m *Manager) GetRelevantSkills(context string) []*Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var relevant []*Skill
	contextLower := strings.ToLower(context)

	for _, skill := range m.skills {
		if skill.MatchPattern != "" && strings.Contains(contextLower, strings.ToLower(skill.MatchPattern)) {
			relevant = append(relevant, skill)
			continue
		}

		nameLower := strings.ToLower(skill.Name)
		if strings.Contains(contextLower, nameLower) {
			relevant = append(relevant, skill)
		}
	}

	return relevant
}

func (m *Manager) GetSystemPromptContext() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n## Available Skills\n\n")
	sb.WriteString("The following specialized skills are loaded and available:\n\n")

	for _, skill := range m.skills {
		sb.WriteString(fmt.Sprintf("### %s\n", skill.Name))
		sb.WriteString(fmt.Sprintf("Description: %s\n\n", skill.Description))
		sb.WriteString(skill.Content)
		sb.WriteString("\n\n---\n\n")
	}

	return sb.String()
}

func (m *Manager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.skills[name]; !ok {
		return fmt.Errorf("skill not found: %s", name)
	}

	delete(m.skills, name)
	return nil
}
