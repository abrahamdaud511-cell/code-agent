package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name      string
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Danger    lipgloss.Color
	Info      lipgloss.Color
	Muted     lipgloss.Color
	Text      lipgloss.Color
	TextDim   lipgloss.Color
	BG        lipgloss.Color
	BGDim     lipgloss.Color
	Border    lipgloss.Color
	Selection lipgloss.Color
	Error     lipgloss.Color
	Accent    lipgloss.Color
}

var Themes = map[string]Theme{
	"catppuccin-mocha": {
		Name: "Catppuccin Mocha", Primary: "#89b4fa", Secondary: "#a6e3a1",
		Success: "#a6e3a1", Warning: "#f9e2af", Danger: "#f38ba8",
		Info: "#89b4fa", Muted: "#585b70", Text: "#cdd6f4", TextDim: "#6c7086",
		BG: "#1e1e2e", BGDim: "#181825", Border: "#313244", Selection: "#45475a",
		Error: "#f38ba8", Accent: "#cba6f7",
	},
	"catppuccin-latte": {
		Name: "Catppuccin Latte", Primary: "#1e66f5", Secondary: "#40a02b",
		Success: "#40a02b", Warning: "#df8e1d", Danger: "#d20f39",
		Info: "#1e66f5", Muted: "#9ca0b0", Text: "#4c4f69", TextDim: "#8c8fa1",
		BG: "#eff1f5", BGDim: "#e6e9ef", Border: "#ccd0da", Selection: "#acb0be",
		Error: "#d20f39", Accent: "#8839ef",
	},
	"dracula": {
		Name: "Dracula", Primary: "#bd93f9", Secondary: "#50fa7b",
		Success: "#50fa7b", Warning: "#f1fa8c", Danger: "#ff5555",
		Info: "#8be9fd", Muted: "#6272a4", Text: "#f8f8f2", TextDim: "#6272a4",
		BG: "#282a36", BGDim: "#21222c", Border: "#44475a", Selection: "#44475a",
		Error: "#ff5555", Accent: "#ff79c6",
	},
	"nord": {
		Name: "Nord", Primary: "#88c0d0", Secondary: "#a3be8c",
		Success: "#a3be8c", Warning: "#ebcb8b", Danger: "#bf616a",
		Info: "#81a1c1", Muted: "#4c566a", Text: "#d8dee9", TextDim: "#616e88",
		BG: "#2e3440", BGDim: "#3b4252", Border: "#434c5e", Selection: "#434c5e",
		Error: "#bf616a", Accent: "#b48ead",
	},
	"solarized-dark": {
		Name: "Solarized Dark", Primary: "#268bd2", Secondary: "#859900",
		Success: "#859900", Warning: "#b58900", Danger: "#dc322f",
		Info: "#2aa198", Muted: "#586e75", Text: "#93a1a1", TextDim: "#657b83",
		BG: "#002b36", BGDim: "#073642", Border: "#073642", Selection: "#073642",
		Error: "#dc322f", Accent: "#6c71c4",
	},
	"solarized-light": {
		Name: "Solarized Light", Primary: "#268bd2", Secondary: "#859900",
		Success: "#859900", Warning: "#b58900", Danger: "#dc322f",
		Info: "#2aa198", Muted: "#93a1a1", Text: "#657b83", TextDim: "#839496",
		BG: "#fdf6e3", BGDim: "#eee8d5", Border: "#eee8d5", Selection: "#eee8d5",
		Error: "#dc322f", Accent: "#6c71c4",
	},
	"gruvbox": {
		Name: "Gruvbox", Primary: "#458588", Secondary: "#98971a",
		Success: "#98971a", Warning: "#d79921", Danger: "#cc241d",
		Info: "#458588", Muted: "#928374", Text: "#ebdbb2", TextDim: "#a89984",
		BG: "#282828", BGDim: "#1d2021", Border: "#3c3836", Selection: "#504945",
		Error: "#cc241d", Accent: "#b16286",
	},
	"monokai": {
		Name: "Monokai", Primary: "#66d9ef", Secondary: "#a6e22e",
		Success: "#a6e22e", Warning: "#e6db74", Danger: "#f92672",
		Info: "#66d9ef", Muted: "#75715e", Text: "#f8f8f2", TextDim: "#75715e",
		BG: "#272822", BGDim: "#1e1f1c", Border: "#3e3d32", Selection: "#3e3d32",
		Error: "#f92672", Accent: "#ae81ff",
	},
	"one-dark": {
		Name: "One Dark", Primary: "#61afef", Secondary: "#98c379",
		Success: "#98c379", Warning: "#e5c07b", Danger: "#e06c75",
		Info: "#56b6c2", Muted: "#5c6370", Text: "#abb2bf", TextDim: "#5c6370",
		BG: "#282c34", BGDim: "#21252b", Border: "#3e4452", Selection: "#3e4452",
		Error: "#e06c75", Accent: "#c678dd",
	},
	"tokyo-night": {
		Name: "Tokyo Night", Primary: "#7aa2f7", Secondary: "#9ece6a",
		Success: "#9ece6a", Warning: "#e0af68", Danger: "#f7768e",
		Info: "#2ac3de", Muted: "#565f89", Text: "#a9b1d6", TextDim: "#565f89",
		BG: "#1a1b26", BGDim: "#16161e", Border: "#24283b", Selection: "#2f3b54",
		Error: "#f7768e", Accent: "#bb9af7",
	},
}

func GetTheme(name string) Theme {
	if t, ok := Themes[name]; ok {
		return t
	}
	return Themes["catppuccin-mocha"]
}
