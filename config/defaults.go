package config

const DefaultTUIConfig = `{
  "theme": "catppuccin-mocha",
  "keybinds": {
    "ctrl+c": "cancel",
    "ctrl+d": "exit",
    "ctrl+l": "clear",
    "ctrl+x": "leader",
    "ctrl+p": "command-palette",
    "ctrl+n": "new-session",
    "tab": "toggle-mode",
    "ctrl+z": "undo",
    "ctrl+y": "redo"
  },
  "scroll_speed": 3,
  "scroll_acceleration": {
    "enabled": true
  },
  "diff_style": "auto",
  "mouse": true,
  "attention": {
    "enabled": false,
    "notifications": true,
    "sound": false,
    "volume": 0.4,
    "sound_pack": "codeagent.default"
  }
}`

const DefaultConfigJSON = `{
  "default_model": "openai/gpt-5",
  "default_provider": "openai",
  "theme": "catppuccin-mocha",
  "log_level": "INFO",
  "permissions": [
    { "tool": "bash", "action": "ask" },
    { "tool": "read", "action": "allow" },
    { "tool": "edit", "action": "ask" },
    { "tool": "write", "action": "ask" },
    { "tool": "grep", "action": "allow" },
    { "tool": "glob", "action": "allow" },
    { "tool": "webfetch", "action": "allow" },
    { "tool": "websearch", "action": "ask" },
    { "tool": "question", "action": "allow" }
  ]
}`
