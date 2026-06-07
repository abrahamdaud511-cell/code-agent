package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"codeagent/core/crypto"
)

type Config struct {
	// Core settings
	DataDir    string `json:"data_dir"`
	ConfigDir  string `json:"config_dir"`
	CacheDir   string `json:"cache_dir"`
	LogLevel   string `json:"log_level"`
	PrintLogs  bool   `json:"print_logs"`
	PureMode   bool   `json:"pure_mode"`

	// Provider settings
	Providers       map[string]ProviderConfig `json:"providers"`
	DefaultProvider string                    `json:"default_provider"`
	DefaultModel    string                    `json:"default_model"`

	// Agent settings
	DefaultAgent string        `json:"default_agent"`
	Agents       []AgentConfig `json:"agents"`

	// TUI settings
	Theme        string       `json:"theme"`
	TUIConfig    TUIConfig    `json:"tui_config,omitempty"`

	// Plugin settings
	PluginDir string `json:"plugin_dir,omitempty"`
	Plugins   []PluginConfig `json:"plugins,omitempty"`

	// Skill settings
	SkillDir string `json:"skill_dir,omitempty"`

	// MCP servers
	MCPServers map[string]MCPServerConfig `json:"mcp_servers"`

	// Permission rules
	Permissions []PermissionRule `json:"permissions"`

	// Filesystem scoping
	AllowedPaths []string `json:"allowed_paths"`
	DeniedPaths  []string `json:"denied_paths"`

	// Environment variable references
	Env map[string]string `json:"env"`

	// Credential store (not serialized)
	credStore *crypto.CredentialStore `json:"-"`
}

type TUIConfig struct {
	Theme       string                 `json:"theme,omitempty"`
	Keybinds    map[string]string      `json:"keybinds,omitempty"`
	ScrollSpeed int                    `json:"scroll_speed,omitempty"`
	Mouse       bool                   `json:"mouse,omitempty"`
	DiffStyle   string                 `json:"diff_style,omitempty"`
	Attention   AttentionConfig        `json:"attention,omitempty"`
	Plugins     []string               `json:"plugins,omitempty"`
}

type AttentionConfig struct {
	Enabled      bool   `json:"enabled"`
	Notifications bool  `json:"notifications"`
	Sound        bool   `json:"sound"`
	Volume       float64 `json:"volume"`
	SoundPack    string `json:"sound_pack"`
}

type PluginConfig struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Enabled bool   `json:"enabled"`
}

type ProviderConfig struct {
	APIKey  string   `json:"api_key"`
	BaseURL string   `json:"base_url"`
	Models  []string `json:"models"`
}

type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

type PermissionRule struct {
	Tool   string `json:"tool"`
	Action string `json:"action"`
}

type AgentConfig struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Model       string   `json:"model"`
	Mode        string   `json:"mode"`
	Permissions []string `json:"permissions"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		cacheHome = filepath.Join(home, ".cache")
	}

	codeagentDataDir := filepath.Join(dataHome, "codeagent")

	return &Config{
		DataDir:    codeagentDataDir,
		ConfigDir:  filepath.Join(configHome, "codeagent"),
		CacheDir:   filepath.Join(cacheHome, "codeagent"),
		LogLevel:   "INFO",
		PrintLogs:  false,
		PureMode:   false,
		Providers:  make(map[string]ProviderConfig),
		MCPServers: make(map[string]MCPServerConfig),
		Agents:     make([]AgentConfig, 0),
		Permissions: []PermissionRule{
			{Tool: "bash", Action: "ask"},
			{Tool: "read", Action: "allow"},
			{Tool: "edit", Action: "ask"},
			{Tool: "write", Action: "ask"},
			{Tool: "grep", Action: "allow"},
			{Tool: "glob", Action: "allow"},
			{Tool: "webfetch", Action: "allow"},
			{Tool: "websearch", Action: "ask"},
			{Tool: "question", Action: "allow"},
			{Tool: "task", Action: "ask"},
			{Tool: "todowrite", Action: "ask"},
		},
		DefaultModel: "openai/gpt-5",
		Theme:        "catppuccin-mocha",
		TUIConfig: TUIConfig{
			Theme:       "catppuccin-mocha",
			ScrollSpeed: 3,
			Mouse:       true,
			DiffStyle:   "auto",
			Keybinds: map[string]string{
				"ctrl+c": "cancel",
				"ctrl+d": "exit",
				"ctrl+l": "clear",
				"ctrl+p": "command-palette",
				"ctrl+n": "new-session",
				"tab":    "toggle-mode",
				"ctrl+z": "undo",
				"ctrl+y": "redo",
			},
			Attention: AttentionConfig{
				Enabled:      false,
				Notifications: true,
				Sound:        false,
				Volume:       0.4,
				SoundPack:    "codeagent.default",
			},
		},
		PluginDir:    filepath.Join(configHome, "codeagent", "plugins"),
		SkillDir:     filepath.Join(configHome, "codeagent", "skills"),
		Plugins:      make([]PluginConfig, 0),
		AllowedPaths: []string{},
		DeniedPaths:  []string{},
		Env:          make(map[string]string),
		credStore:    crypto.NewCredentialStore(codeagentDataDir),
	}
}

func (c *Config) GetCredentialStore() *crypto.CredentialStore {
	return c.credStore
}

func Load(cfgPath string) (*Config, error) {
	cfg := DefaultConfig()

	if cfgPath != "" {
		if err := cfg.loadFromFile(cfgPath); err != nil {
			return nil, err
		}
	} else {
		paths := cfg.configPaths()
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				if err := cfg.loadFromFile(p); err != nil {
					return nil, err
				}
			}
		}
	}

	// Load credentials from encrypted store
	cfg.loadCredentials()

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	// Create data directories
	for _, dir := range []string{cfg.DataDir, cfg.ConfigDir, cfg.CacheDir} {
		os.MkdirAll(dir, 0755)
	}

	// Create plugin and skill directories
	if cfg.PluginDir != "" {
		os.MkdirAll(cfg.PluginDir, 0755)
	}
	if cfg.SkillDir != "" {
		os.MkdirAll(cfg.SkillDir, 0755)
	}

	return cfg, nil
}

func (c *Config) configPaths() []string {
	home, _ := os.UserHomeDir()
	paths := []string{}

	paths = append(paths, filepath.Join(home, ".config", "codeagent", "codeagent.json"))

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		paths = append(paths, filepath.Join(xdg, "codeagent", "codeagent.json"))
	}

	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "codeagent.json"))
		paths = append(paths, filepath.Join(cwd, ".codeagent", "codeagent.json"))
		paths = append(paths, filepath.Join(cwd, ".codeagent.json"))
	}

	if runtime.GOOS != "windows" {
		paths = append(paths, "/etc/codeagent/codeagent.json")
	}

	return paths
}

func (c *Config) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := stripComments(string(data))

	var fileCfg Config
	if err := json.Unmarshal([]byte(content), &fileCfg); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	c.merge(&fileCfg)
	return nil
}

func (c *Config) merge(other *Config) {
	if other.DataDir != "" {
		c.DataDir = other.DataDir
	}
	if other.ConfigDir != "" {
		c.ConfigDir = other.ConfigDir
	}
	if other.CacheDir != "" {
		c.CacheDir = other.CacheDir
	}
	if other.LogLevel != "" {
		c.LogLevel = other.LogLevel
	}
	if other.DefaultProvider != "" {
		c.DefaultProvider = other.DefaultProvider
	}
	if other.DefaultModel != "" {
		c.DefaultModel = other.DefaultModel
	}
	if other.DefaultAgent != "" {
		c.DefaultAgent = other.DefaultAgent
	}
	if other.Theme != "" {
		c.Theme = other.Theme
	}
	if other.PrintLogs {
		c.PrintLogs = other.PrintLogs
	}
	if other.PureMode {
		c.PureMode = other.PureMode
	}
	if other.PluginDir != "" {
		c.PluginDir = other.PluginDir
	}
	if other.SkillDir != "" {
		c.SkillDir = other.SkillDir
	}
	for k, v := range other.Providers {
		c.Providers[k] = v
	}
	for k, v := range other.MCPServers {
		c.MCPServers[k] = v
	}
	for _, rule := range other.Permissions {
		c.Permissions = append(c.Permissions, rule)
	}
	c.Agents = append(c.Agents, other.Agents...)
	c.AllowedPaths = append(c.AllowedPaths, other.AllowedPaths...)
	c.DeniedPaths = append(c.DeniedPaths, other.DeniedPaths...)
	for k, v := range other.Env {
		c.Env[k] = v
	}

	// Merge TUIConfig
	tuiJSON, _ := json.Marshal(other.TUIConfig)
	if string(tuiJSON) != "{}" {
		if other.TUIConfig.Theme != "" {
			c.TUIConfig.Theme = other.TUIConfig.Theme
		}
		if other.TUIConfig.ScrollSpeed > 0 {
			c.TUIConfig.ScrollSpeed = other.TUIConfig.ScrollSpeed
		}
		if other.TUIConfig.DiffStyle != "" {
			c.TUIConfig.DiffStyle = other.TUIConfig.DiffStyle
		}
		if other.TUIConfig.Keybinds != nil {
			if c.TUIConfig.Keybinds == nil {
				c.TUIConfig.Keybinds = make(map[string]string)
			}
			for k, v := range other.TUIConfig.Keybinds {
				c.TUIConfig.Keybinds[k] = v
			}
		}
		if other.TUIConfig.Plugins != nil {
			c.TUIConfig.Plugins = other.TUIConfig.Plugins
		}
	}
}

func (c *Config) loadCredentials() {
	creds, err := c.credStore.LoadCredentials()
	if err == nil {
		for provider, apiKey := range creds {
			if existing, ok := c.Providers[provider]; ok {
				if existing.APIKey == "" {
					existing.APIKey = apiKey
					c.Providers[provider] = existing
				}
			} else {
				c.Providers[provider] = ProviderConfig{APIKey: apiKey}
			}
		}
	}
}

func (c *Config) applyEnvOverrides() {
	envVars := map[string]string{
		"CODEAGENT_DATA_DIR":       "data_dir",
		"CODEAGENT_CONFIG_DIR":     "config_dir",
		"CODEAGENT_CACHE_DIR":      "cache_dir",
		"CODEAGENT_LOG_LEVEL":      "log_level",
		"CODEAGENT_DEFAULT_MODEL":  "default_model",
		"CODEAGENT_THEME":          "theme",
		"CODEAGENT_PLUGIN_DIR":     "plugin_dir",
		"CODEAGENT_SKILL_DIR":      "skill_dir",
	}

	for envKey, cfgKey := range envVars {
		if val := os.Getenv(envKey); val != "" {
			switch cfgKey {
			case "data_dir":
				c.DataDir = val
			case "config_dir":
				c.ConfigDir = val
			case "cache_dir":
				c.CacheDir = val
			case "log_level":
				c.LogLevel = val
			case "default_model":
				c.DefaultModel = val
			case "theme":
				c.Theme = val
			case "plugin_dir":
				c.PluginDir = val
			case "skill_dir":
				c.SkillDir = val
			}
		}
	}

	providerEnvVars := map[string]string{
		"OPENAI_API_KEY":           "openai",
		"ANTHROPIC_API_KEY":        "anthropic",
		"GOOGLE_API_KEY":           "google",
		"GROQ_API_KEY":             "groq",
		"OPENROUTER_API_KEY":       "openrouter",
		"MISTRAL_API_KEY":          "mistral",
		"DEEPSEEK_API_KEY":         "deepseek",
		"COHERE_API_KEY":           "cohere",
		"PERPLEXITY_API_KEY":       "perplexity",
		"XAI_API_KEY":              "xai",
		"TOGETHERAI_API_KEY":       "togetherai",
		"DEEPINFRA_API_KEY":        "deepinfra",
		"CEREBRAS_API_KEY":         "cerebras",
		"ALIBABA_API_KEY":          "alibaba",
		"VENICE_API_KEY":           "venice",
		"AZURE_OPENAI_API_KEY":     "azure",
		"AWS_ACCESS_KEY_ID":        "aws-bedrock",
		"GITHUB_TOKEN":             "github-copilot",
		"GITHUB_COPILOT_TOKEN":     "github-copilot",
	}

	for envKey, provider := range providerEnvVars {
		if val := os.Getenv(envKey); val != "" {
			if p, ok := c.Providers[provider]; ok {
				if p.APIKey == "" {
					p.APIKey = val
					c.Providers[provider] = p
				}
			} else {
				c.Providers[provider] = ProviderConfig{APIKey: val}
			}
		}
	}
}

func (c *Config) AddMCPServer(name, command string, args []string) {
	if c.MCPServers == nil {
		c.MCPServers = make(map[string]MCPServerConfig)
	}
	c.MCPServers[name] = MCPServerConfig{
		Command: command,
		Args:    args,
	}
}

func (c *Config) Save() error {
	configFile := filepath.Join(c.ConfigDir, "codeagent.json")
	os.MkdirAll(c.ConfigDir, 0755)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func (c *Config) ProviderNames() []string {
	names := make([]string, 0, len(c.Providers))
	for name := range c.Providers {
		names = append(names, name)
	}
	return names
}

func (c *Config) IsPathAllowed(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Check denied paths first
	for _, denied := range c.DeniedPaths {
		deniedAbs, err := filepath.Abs(denied)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, deniedAbs) {
			return false
		}
	}

	// If allowed paths specified, restrict to those
	if len(c.AllowedPaths) > 0 {
		for _, allowed := range c.AllowedPaths {
			allowedAbs, err := filepath.Abs(allowed)
			if err != nil {
				continue
			}
			if strings.HasPrefix(absPath, allowedAbs) {
				return true
			}
		}
		return false
	}

	return true
}

func stripComments(content string) string {
	var result strings.Builder
	inString := false
	prevChar := byte(0)
	i := 0

	for i < len(content) {
		ch := content[i]

		if inString {
			if ch == '"' && prevChar != '\\' {
				inString = false
			}
			result.WriteByte(ch)
			prevChar = ch
			i++
			continue
		}

		if ch == '"' {
			inString = true
			result.WriteByte(ch)
			prevChar = ch
			i++
			continue
		}

		if ch == '/' && i+1 < len(content) {
			next := content[i+1]
			if next == '/' {
				i += 2
				for i < len(content) && content[i] != '\n' {
					i++
				}
				continue
			}
			if next == '*' {
				i += 2
				for i+1 < len(content) && !(content[i] == '*' && content[i+1] == '/') {
					i++
				}
				i += 2
				continue
			}
		}

		result.WriteByte(ch)
		prevChar = ch
		i++
	}

	return result.String()
}
