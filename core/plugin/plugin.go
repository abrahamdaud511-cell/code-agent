package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"codeagent/core/bus"
)

type Plugin interface {
	Name() string
	Version() string
	Description() string
	Init() error
	Shutdown() error
}

type TUIExtension interface {
	Plugin
	Render() string
	HandleInput(input string) (string, bool)
}

type ToolExtension interface {
	Plugin
	ToolName() string
	ToolDefinition() map[string]interface{}
	Execute(args map[string]interface{}) (string, error)
}

type Manager struct {
	mu       sync.RWMutex
	plugins  map[string]Plugin
	pluginDir string
}

func NewManager(pluginDir string) *Manager {
	return &Manager{
		plugins:   make(map[string]Plugin),
		pluginDir: pluginDir,
	}
}

func (m *Manager) LoadAll() error {
	if m.pluginDir == "" {
		return nil
	}

	if _, err := os.Stat(m.pluginDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".so" && ext != ".wasm" {
			continue
		}

		path := filepath.Join(m.pluginDir, name)
		if err := m.Load(path); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load plugin %s: %v\n", name, err)
		}
	}

	return nil
}

func (m *Manager) Load(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	symPlugin, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin does not export 'Plugin' symbol: %w", err)
	}

	plug, ok := symPlugin.(Plugin)
	if !ok {
		return fmt.Errorf("plugin symbol does not implement Plugin interface")
	}

	if err := plug.Init(); err != nil {
		return fmt.Errorf("plugin init failed: %w", err)
	}

	m.plugins[plug.Name()] = plug

	bus.PublishEvent(bus.EventPluginLoad, map[string]string{
		"name":    plug.Name(),
		"version": plug.Version(),
	}, "plugin")

	return nil
}

func (m *Manager) Unload(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plug, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	if err := plug.Shutdown(); err != nil {
		return fmt.Errorf("plugin shutdown failed: %w", err)
	}

	delete(m.plugins, name)

	bus.PublishEvent(bus.EventPluginUnload, map[string]string{
		"name": name,
	}, "plugin")

	return nil
}

func (m *Manager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

func (m *Manager) List() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		list = append(list, p)
	}
	return list
}

func (m *Manager) ListTUIExtensions() []TUIExtension {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var extensions []TUIExtension
	for _, p := range m.plugins {
		if ext, ok := p.(TUIExtension); ok {
			extensions = append(extensions, ext)
		}
	}
	return extensions
}

func (m *Manager) ListToolExtensions() []ToolExtension {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var extensions []ToolExtension
	for _, p := range m.plugins {
		if ext, ok := p.(ToolExtension); ok {
			extensions = append(extensions, ext)
		}
	}
	return extensions
}

func (m *Manager) ShutdownAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, plug := range m.plugins {
		plug.Shutdown()
		delete(m.plugins, name)
	}
}
