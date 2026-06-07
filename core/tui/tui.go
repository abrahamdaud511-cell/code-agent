package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"codeagent/core/agent"
	"codeagent/core/bus"
	"codeagent/config"
	"codeagent/providers"
	"codeagent/core/session"
)

const logo = `
  ╔══════════════════════════════════════════════╗
  ║   ██████╗ ██████╗ ██████╗ ███████╗ █████╗   ║
  ║  ██╔════╝██╔═══██╗██╔══██╗██╔════╝██╔══██╗  ║
  ║  ██║     ██║   ██║██║  ██║█████╗  ███████║  ║
  ║  ██║     ██║   ██║██║  ██║██╔══╝  ██╔══██║  ║
  ║  ╚██████╗╚██████╔╝██████╔╝███████╗██║  ██║  ║
  ║   ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝  ║
  ║           AI Coding Agent v1.0.0             ║
  ╚══════════════════════════════════════════════╝
`

type App struct {
	cfg          *config.Config
	session      *session.Session
	sessionStore *session.Store
	ctx          context.Context
	cancel       context.CancelFunc
	theme        Theme
	renderer     *glamour.TermRenderer
}

type Model struct {
	app          *App
	ready        bool
	viewport     viewport.Model
	input        textinput.Model
	spinner      spinner.Model
	loading      bool
	messages     []chatMessage
	width        int
	height       int
	mode         string
	showHelp     bool
	showDetails  bool
	showThinking bool
	showPalette  bool
	paletteItems []paletteItem
	paletteIdx   int
	thinking     []string
	err          error

	// Theme
	theme Theme

	// Cursor and selection
	cursor int

	// Renderers
	renderer     *glamour.TermRenderer
	rendererDark *glamour.TermRenderer

	// Style cache
	styles map[string]lipgloss.Style
}

type chatMessage struct {
	Role      string
	Content   string
	Time      time.Time
	ToolCalls []ToolCallInfo
}

type ToolCallInfo struct {
	Tool   string
	Input  string
	Result string
	Status string
}

type paletteItem struct {
	Key         string
	Description string
	Action      string
	Category    string
}

type responseMsg struct {
	content   string
	toolCalls []ToolCallInfo
}

type errorMsg struct {
	err string
}

var (
	modeColors = map[string]string{
		"build": "#a6e3a1", "plan": "#f9e2af",
		"debug": "#f38ba8", "review": "#89b4fa", "docs": "#cba6f7",
	}
	modeLabels = map[string]string{
		"build": "BUILD", "plan": "PLAN", "debug": "DEBUG",
		"review": "REVIEW", "docs": "DOCS",
	}
	modeDescs = map[string]string{
		"build": "Full access — read, write, execute",
		"plan":  "Read-only — analyze and plan",
		"debug": "Investigation — find bugs",
		"review": "Code review — quality check",
		"docs":  "Documentation — write docs",
	}
)

func New(cfg *config.Config, sess *session.Session, store *session.Store) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())
	theme := GetTheme(cfg.Theme)

	ren, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)

	return &App{
		cfg:          cfg,
		session:      sess,
		sessionStore: store,
		ctx:          ctx,
		cancel:       cancel,
		theme:        theme,
		renderer:     ren,
	}, nil
}

func (a *App) SetInitialPrompt(p string) {}

func (a *App) Run() (tea.Model, error) {
	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(a.theme.Primary))),
	)

	ti := textinput.New()
	ti.Placeholder = " Ask CodeAgent anything... (@file, !shell, /help, Ctrl+P)"
	ti.Focus()
	ti.CharLimit = 4000
	ti.Width = 80
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(a.theme.Primary)).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(a.theme.Text))

	ren, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)

	m := &Model{
		app:          a,
		spinner:      s,
		input:        ti,
		messages:     make([]chatMessage, 0),
		mode:         "build",
		showHelp:     false,
		showPalette:  false,
		thinking:     make([]string, 0),
		renderer:     ren,
		theme:        a.theme,
		styles:       make(map[string]lipgloss.Style),
		paletteItems: defaultPaletteItems(),
		cursor:       0,
	}

	// Load existing session messages
	for _, msg := range a.session.GetHistory() {
		if msg.Role != "system" && msg.Content != "" {
			m.messages = append(m.messages, chatMessage{
				Role: msg.Role, Content: msg.Content, Time: msg.Timestamp,
			})
		}
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	return p.Run()
}

func defaultPaletteItems() []paletteItem {
	return []paletteItem{
		{Key: "/help", Description: "Show help screen", Action: "help", Category: "Help"},
		{Key: "/exit", Description: "Exit CodeAgent", Action: "exit", Category: "Session"},
		{Key: "/new", Description: "Start a fresh session", Action: "new", Category: "Session"},
		{Key: "/mode build", Description: "Switch to Build mode", Action: "mode", Category: "Mode"},
		{Key: "/mode plan", Description: "Switch to Plan mode", Action: "mode", Category: "Mode"},
		{Key: "/mode debug", Description: "Switch to Debug mode", Action: "mode", Category: "Mode"},
		{Key: "/mode review", Description: "Switch to Review mode", Action: "mode", Category: "Mode"},
		{Key: "/mode docs", Description: "Switch to Docs mode", Action: "mode", Category: "Mode"},
		{Key: "/model", Description: "Change AI model", Action: "model", Category: "Config"},
		{Key: "/models", Description: "List available models", Action: "models", Category: "Config"},
		{Key: "/connect", Description: "Add provider API key", Action: "connect", Category: "Config"},
		{Key: "/init", Description: "Create AGENTS.md", Action: "init", Category: "Project"},
		{Key: "/undo", Description: "Revert last changes", Action: "undo", Category: "Session"},
		{Key: "/redo", Description: "Restore undone changes", Action: "redo", Category: "Session"},
		{Key: "/sessions", Description: "List saved sessions", Action: "sessions", Category: "Session"},
		{Key: "/compact", Description: "Summarize context", Action: "compact", Category: "Session"},
		{Key: "/export", Description: "Save conversation", Action: "export", Category: "Session"},
		{Key: "/share", Description: "Share session link", Action: "share", Category: "Session"},
		{Key: "/details", Description: "Toggle tool details", Action: "details", Category: "Display"},
		{Key: "/thinking", Description: "Toggle thinking display", Action: "thinking", Category: "Display"},
		{Key: "/permission", Description: "View permission rules", Action: "permission", Category: "Config"},
		{Key: "/themes", Description: "List available themes", Action: "themes", Category: "Display"},
		{Key: "/editor", Description: "Open external editor", Action: "editor", Category: "Tools"},
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-7)
			m.viewport.YPosition = 0
			m.input.Width = msg.Width - 4
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 7
		}

	case tea.KeyMsg:
		if m.showPalette {
			return m.handlePaletteKey(msg)
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			m.addMsg("system", "Goodbye! 👋")
			bus.PublishEvent(bus.EventUserInput, map[string]string{"action": "exit"}, "tui")
			return m, tea.Quit

		case tea.KeyEscape:
			m.showHelp = false

		case tea.KeyEnter:
			if m.loading {
				break
			}
			input := strings.TrimSpace(m.input.Value())
			if input == "" {
				break
			}
			m.input.SetValue("")

			if strings.HasPrefix(input, "/") {
				return m, m.handleCommand(input)
			}

			if strings.HasPrefix(input, "!") {
				return m, m.execBash(strings.TrimPrefix(input, "!"))
			}

			input = m.processFileRefs(input)

			m.messages = append(m.messages, chatMessage{
				Role: "user", Content: input, Time: time.Now(),
			})
			m.loading = true
			m.cursor = len(m.messages)
			m.viewport.GotoBottom()
			return m, m.processInput(input)

		case tea.KeyTab:
			modes := []string{"build", "plan", "debug", "review", "docs"}
			for i, mode := range modes {
				if m.mode == mode {
					m.mode = modes[(i+1)%len(modes)]
					m.app.session.SetAgent(m.mode)
					bus.PublishEvent(bus.EventModeChange, map[string]string{
						"from": modes[i], "to": m.mode,
					}, "tui")
					break
				}
			}
			m.addMsg("system", " Switched to "+m.style("prim", strings.ToUpper(m.mode)).Bold(true).Render()+" mode — "+m.modeDesc())

		case tea.KeyCtrlP:
			m.showPalette = !m.showPalette
			m.paletteIdx = 0

		case tea.KeyCtrlN:
			m.messages = make([]chatMessage, 0)
			m.app.session = session.New(m.app.cfg, m.app.sessionStore)
			m.addMsg("system", "New session started")

		case tea.KeyCtrlZ:
			return m, m.handleCommand("/undo")

		case tea.KeyCtrlY:
			return m, m.handleCommand("/redo")

		case tea.KeyCtrlL:
			m.messages = make([]chatMessage, 0)
			m.addMsg("system", "Display cleared")

		case tea.KeyUp:
			m.viewport.LineUp(3)
		case tea.KeyDown:
			m.viewport.LineDown(3)
		case tea.KeyPgUp:
			m.viewport.HalfViewUp()
		case tea.KeyPgDown:
			m.viewport.HalfViewDown()
		}

	case responseMsg:
		m.loading = false
		m.messages = append(m.messages, chatMessage{
			Role: "assistant", Content: msg.content, Time: time.Now(),
			ToolCalls: msg.toolCalls,
		})
		m.cursor = len(m.messages)
		m.viewport.GotoBottom()

	case errorMsg:
		m.loading = false
		m.addMsg("system", m.style("err", " Error: "+msg.err).Render())

	case bus.Event:
		// Handle bus events in TUI
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) handlePaletteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape, tea.KeyCtrlP:
		m.showPalette = false
		return m, nil

	case tea.KeyEnter:
		if m.paletteIdx >= 0 && m.paletteIdx < len(m.paletteItems) {
			item := m.paletteItems[m.paletteIdx]
			m.showPalette = false
			m.input.SetValue(item.Key)
			return m, m.handleCommand(item.Key)
		}

	case tea.KeyUp:
		if m.paletteIdx > 0 {
			m.paletteIdx--
		}

	case tea.KeyDown:
		if m.paletteIdx < len(m.paletteItems)-1 {
			m.paletteIdx++
		}

	case tea.KeyTab:
		if m.paletteIdx < len(m.paletteItems)-1 {
			m.paletteIdx++
		}

	default:
		// Handle text input for filtering
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace {
			m.input, _ = m.input.Update(msg)
			m.paletteIdx = 0
		}
	}

	return m, nil
}

func (m *Model) View() string {
	if !m.ready {
		return "\n\n" + m.style("prim", logo).Render() + "\n\n" +
			m.style("dim", " Loading...").Render() + m.spinner.View()
	}

	if m.showPalette {
		return m.renderPalette()
	}

	var buf strings.Builder

	if len(m.messages) == 0 && !m.showHelp {
		buf.WriteString(m.style("prim", logo).Render())
		buf.WriteString("\n")
		buf.WriteString(m.style("text", " Welcome to CodeAgent! ").Bold(true).Render())
		buf.WriteString(m.style("dim", "v1.0.0").Render())
		buf.WriteString("\n")
		buf.WriteString(m.style("dim", " Type ").Render() +
			m.style("succ", "/help").Render() +
			m.style("dim", " for commands, ").Render() +
			m.style("succ", "Tab").Render() +
			m.style("dim", " for modes, ").Render() +
			m.style("succ", "@file").Render() +
			m.style("dim", " to reference files").Render())
		buf.WriteString("\n\n")
	}

	var vb strings.Builder
	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			vb.WriteString(m.style("prim", " ┌─ You").Bold(true).Render())
			vb.WriteString(m.style("dim", " ("+msg.Time.Format("15:04")+")").Render())
			vb.WriteString("\n")
			lines := strings.Split(msg.Content, "\n")
			for _, line := range lines {
				vb.WriteString(" │ ")
				vb.WriteString(m.style("text", line).Render())
				vb.WriteString("\n")
			}
			vb.WriteString(m.style("prim", " └─\n\n").Render())

		case "assistant":
			vb.WriteString(m.style("succ", " ┌─ CodeAgent").Bold(true).Render())
			vb.WriteString(m.style("dim", " ("+msg.Time.Format("15:04")+")").Render())
			if m.mode != "build" {
				vb.WriteString(" " + m.style("accent", "["+strings.ToUpper(m.mode)+"]").Render())
			}
			vb.WriteString("\n")

			for _, tc := range msg.ToolCalls {
				statusStyle := m.style("succ", "✓")
				if tc.Status == "error" {
					statusStyle = m.style("err", "✗")
				}
				vb.WriteString(" │ ")
				vb.WriteString(m.style("dim", "🔧 "+tc.Tool+" ").Render())
				vb.WriteString(statusStyle.Render())
				vb.WriteString("\n")
				if m.showDetails {
					vb.WriteString(" │   " + m.style("dim", "Input: "+tc.Input).Render() + "\n")
					if tc.Result != "" {
						vb.WriteString(" │   " + m.style("dim", "Result: "+truncate(tc.Result, 200)).Render() + "\n")
					}
				}
			}

			rendered, err := m.renderer.Render(msg.Content)
			if err == nil {
				lines := strings.Split(strings.TrimSpace(rendered), "\n")
				for _, line := range lines {
					vb.WriteString(" │ ")
					vb.WriteString(line)
					vb.WriteString("\n")
				}
			} else {
				lines := strings.Split(msg.Content, "\n")
				for _, line := range lines {
					vb.WriteString(" │ ")
					vb.WriteString(m.style("text", line).Render())
					vb.WriteString("\n")
				}
			}
			vb.WriteString(m.style("succ", " └─\n\n").Render())

		case "system":
			vb.WriteString(m.style("dim", " ── "+msg.Content+"\n\n").Render())
		}
	}

	m.viewport.SetContent(vb.String())
	buf.WriteString(m.viewport.View())

	if m.loading {
		buf.WriteString("\n  ")
		buf.WriteString(m.spinner.View())
		buf.WriteString(m.style("dim", " Thinking...").Render())
		if len(m.thinking) > 0 && m.showThinking {
			buf.WriteString("\n  " + m.style("dim", m.thinking[len(m.thinking)-1]).Render())
		}
		buf.WriteString("\n")
	} else {
		buf.WriteString("\n")
	}

	buf.WriteString(" ")
	buf.WriteString(m.input.View())
	buf.WriteString("\n")

	// Status bar
	modeColor := m.style("succ", "")
	if c, ok := modeColors[m.mode]; ok {
		modeColor = lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Bold(true)
	}

	modelName := m.app.session.Model
	if modelName == "" {
		modelName = "not configured"
	}

	var status string
	status += fmt.Sprintf(" %s %s  ", modeColor.Render("◆ "+strings.ToUpper(m.mode)), m.style("dim", "│  Model: "+modelName).Render())

	if m.showDetails {
		status += m.style("dim", "│  Details: ON  ").Render()
	}
	if m.showThinking {
		status += m.style("dim", "│  Thinking: ON  ").Render()
	}
	status += m.style("dim", "│  /help  ").Render()
	status += m.style("dim", "│  Ctrl+P  ").Render()

	statusBar := lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("255")).
		Render(status)
	buf.WriteString(statusBar)

	return buf.String()
}

func (m *Model) renderPalette() string {
	var buf strings.Builder

	filter := strings.TrimSpace(m.input.Value())
	buf.WriteString(m.style("prim", " Command Palette\n").Bold(true).Render())
	buf.WriteString(m.style("dim", " Type to filter, Enter to select, Esc to close\n\n").Render())

	// Filter items
	var filtered []paletteItem
	if filter == "" {
		filtered = m.paletteItems
	} else {
		lowerFilter := strings.ToLower(filter)
		for _, item := range m.paletteItems {
			if strings.Contains(strings.ToLower(item.Key), lowerFilter) ||
				strings.Contains(strings.ToLower(item.Description), lowerFilter) {
				filtered = append(filtered, item)
			}
		}
	}

	// Group by category
	categories := make(map[string][]paletteItem)
	var catOrder []string
	for _, item := range filtered {
		if _, ok := categories[item.Category]; !ok {
			catOrder = append(catOrder, item.Category)
		}
		categories[item.Category] = append(categories[item.Category], item)
	}

	idx := 0
	for _, cat := range catOrder {
		items := categories[cat]
		buf.WriteString(m.style("accent", " "+cat+"\n").Bold(true).Render())

		for _, item := range items {
			style := m.style("text", "")
			prefix := "  "
			if idx == m.paletteIdx {
				style = m.style("prim", "").Bold(true)
				prefix = " →"
				buf.WriteString(m.style("selection", "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n").Render())
			}
			buf.WriteString(fmt.Sprintf("%s %-25s %s\n",
				prefix,
				style.Render(item.Key),
				m.style("dim", item.Description).Render()))
			idx++
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

func (m *Model) style(name, text string) lipgloss.Style {
	key := name
	if s, ok := m.styles[key]; ok {
		if text != "" {
			return s.SetString(text)
		}
		return s
	}

	var s lipgloss.Style
	switch name {
	case "prim":
		s = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Primary))
	case "succ":
		s = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Success))
	case "warn":
		s = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Warning))
	case "err":
		s = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Error))
	case "dim":
		s = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.TextDim))
	case "text":
		s = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Text))
	case "accent":
		s = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Accent))
	case "selection":
		s = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Selection))
	default:
		s = lipgloss.NewStyle()
	}

	m.styles[key] = s
	if text != "" {
		s = s.SetString(text)
	}
	return s
}

func (m *Model) addMsg(role, content string) {
	m.messages = append(m.messages, chatMessage{
		Role: role, Content: content, Time: time.Now(),
	})
}

func (m *Model) modeDesc() string {
	if d, ok := modeDescs[m.mode]; ok {
		return d
	}
	return ""
}

func (m *Model) processFileRefs(input string) string {
	parts := strings.Fields(input)
	for i, part := range parts {
		if strings.HasPrefix(part, "@") && len(part) > 1 {
			ref := part[1:]
			if ref == "agent" || ref == "all" {
				continue
			}
			candidates := findFile(ref)
			if len(candidates) > 0 {
				data, err := os.ReadFile(candidates[0])
				if err == nil {
					parts[i] = fmt.Sprintf("(file: %s)\n```\n%s\n```", candidates[0], string(data))
				}
			}
		}
	}
	return strings.Join(parts, " ")
}

func findFile(name string) []string {
	var results []string
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.Contains(info.Name(), name) || strings.Contains(path, name) {
			results = append(results, path)
		}
		if len(results) >= 5 {
			return fmt.Errorf("enough")
		}
		return nil
	})
	return results
}

func (m *Model) execBash(cmdStr string) tea.Cmd {
	m.addMsg("system", m.style("dim", "Running: $ "+cmdStr).Render())
	m.loading = true
	return func() tea.Msg {
		cmd := exec.Command("sh", "-c", cmdStr)
		out, err := cmd.CombinedOutput()
		output := string(out)
		if err != nil {
			output += "\n" + m.style("err", "Exit: "+err.Error()).Render()
		}
		m.loading = false
		return responseMsg{content: "```bash\n$ " + cmdStr + "\n" + output + "\n```"}
	}
}

func (m *Model) processInput(input string) tea.Cmd {
	return func() tea.Msg {
		provider, err := providers.GetProvider(m.app.cfg, m.app.session.Model)
		if err != nil {
			return errorMsg{err: fmt.Sprintf("No provider configured. Use /connect first.\n%s", err)}
		}

		ag, err := agent.New(m.app.cfg, m.app.session, provider)
		if err != nil {
			return errorMsg{err: fmt.Sprintf("Agent error: %v", err)}
		}
		ag.SetInteractive(true)

		resp, err := ag.Run(input)
		if err != nil {
			return errorMsg{err: fmt.Sprintf("Error: %v", err)}
		}

		tcs := make([]ToolCallInfo, 0)
		for _, tc := range resp.ToolCalls {
			tcs = append(tcs, ToolCallInfo{
				Tool:   tc.Function.Name,
				Input:  tc.Function.Arguments,
				Status: "completed",
			})
		}
		return responseMsg{content: resp.Text, toolCalls: tcs}
	}
}

func (m *Model) handleCommand(input string) tea.Cmd {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}
	cmd := strings.TrimPrefix(strings.ToLower(parts[0]), "/")
	args := parts[1:]

	switch cmd {
	case "help", "h", "?":
		m.showHelp = !m.showHelp
		if m.showHelp {
			m.addMsg("system", m.renderHelp())
		} else {
			m.addMsg("system", "Help closed")
		}

	case "exit", "quit", "q":
		m.addMsg("system", "Goodbye! 👋")
		return tea.Quit

	case "new", "clear":
		m.messages = make([]chatMessage, 0)
		m.app.session = session.New(m.app.cfg, m.app.sessionStore)
		m.addMsg("system", "New session started")

	case "mode":
		if len(args) == 0 {
			m.addMsg("system", "Usage: /mode <build|plan|debug|review|docs>")
			return nil
		}
		mode := strings.ToLower(args[0])
		valid := map[string]bool{"build": true, "plan": true, "debug": true, "review": true, "docs": true}
		if valid[mode] {
			m.mode = mode
			m.app.session.SetAgent(mode)
			m.addMsg("system", "Switched to "+strings.ToUpper(mode)+" mode")
		}

	case "connect", "connct":
		if len(args) == 0 {
			m.addMsg("system", `Connect to a provider:

  Usage: /connect <provider> [api_key]

  Examples:
    /connect openai sk-...
    /connect anthropic sk-ant-...
    /connect ollama
    /connect google AIza...
    /connect groq gsk-...
    /connect perplexity pplx-...

  Supported providers:
    openai, anthropic, google, groq, openrouter,
    ollama, mistral, deepseek, github-copilot,
    aws-bedrock, azure, cohere, perplexity, xai,
    togetherai, deepinfra, cerebras, alibaba, venice`)
			return nil
		}
		return func() tea.Msg {
			msg := doConnect(args, m.app.cfg)
			return responseMsg{content: msg}
		}

	case "model", "m":
		if len(args) == 0 {
			m.addMsg("system", "Current model: "+m.app.session.Model+"\nUsage: /model <provider/model>\nEx: /model openai/gpt-5")
			return nil
		}
		m.app.session.SetModel(args[0])
		m.addMsg("system", "Model set to: "+m.style("succ", args[0]).Render())

	case "models":
		modelsList := `Available Models:

  OpenAI         gpt-5, gpt-4o, gpt-4o-mini, o4, o3
  Anthropic      claude-sonnet-4, claude-haiku-4, claude-opus-4
  Google         gemini-2.5-pro, gemini-2.5-flash
  Groq           llama-4-scout, llama-4-maverick, deepseek-r1
  Ollama         llama4, deepseek-r1, qwen2.5-coder, phi-4
  Mistral        mistral-large-2505, codestral
  DeepSeek       deepseek-chat, deepseek-reasoner
  OpenRouter     auto (routes to best model)
  GitHub Copilot gpt-4o-copilot, claude-sonnet-4-copilot
  AWS Bedrock    claude-sonnet-4, claude-haiku-4, llama-4
  Azure          gpt-4o, gpt-4o-mini, gpt-5
  Cohere         command-r-plus, command-r, command-a
  Perplexity     sonar-pro, sonar
  xAI            grok-3, grok-3-mini
  Together AI    llama-4-17b, deepseek-v3, qwen-2.5-coder
  DeepInfra      llama-4-scout, deepseek-v3
  Cerebras       llama-4-scout
  Alibaba/Qwen   qwen-max, qwen-plus, qwen-turbo
  Venice         llama-4-scout, deepseek-r1

  Use /model <provider/model> to switch`
		m.addMsg("system", modelsList)

	case "init":
		dir, _ := os.Getwd()
		af := filepath.Join(dir, "AGENTS.md")
		if _, err := os.Stat(af); err == nil {
			m.addMsg("system", "AGENTS.md already exists in "+dir)
			return nil
		}
		content := fmt.Sprintf(`# %s

## Tech Stack
- Language: 
- Framework: 
- Build: 

## Commands
- Build: 
- Test: 
- Lint: 
`, filepath.Base(dir))
		os.WriteFile(af, []byte(content), 0644)
		m.addMsg("system", m.style("succ", "Created AGENTS.md in "+dir).Render())

	case "undo", "u":
		m.addMsg("system", m.style("warn", " Undo: reverting last changes...").Render())
		return func() tea.Msg {
			cmd := exec.Command("git", "checkout", ".")
			if err := cmd.Run(); err != nil {
				return responseMsg{content: "Undo requires a Git repository. Run `git init` first."}
			}
			if len(m.messages) >= 2 {
				m.messages = m.messages[:len(m.messages)-2]
			}
			return responseMsg{content: "Undo complete. Changes reverted."}
		}

	case "redo", "r":
		m.addMsg("system", m.style("warn", " Redo: restoring last changes...").Render())

	case "sessions", "resume", "continue":
		sessions, err := m.app.sessionStore.List()
		if err != nil || len(sessions) == 0 {
			m.addMsg("system", "No saved sessions found.")
			return nil
		}
		var sb strings.Builder
		sb.WriteString("Saved Sessions:\n\n")
		for i, s := range sessions {
			title := s.Title
			if len(title) > 50 {
				title = title[:50] + "..."
			}
			sb.WriteString(fmt.Sprintf("  %d. %s  %s\n", i+1, m.style("prim", title).Render(), m.style("dim", s.ID[:12]+"...").Render()))
		}
		sb.WriteString("\nUse: codeagent --session <id>")
		m.addMsg("system", sb.String())

	case "compact", "summarize":
		m.app.session.Compact()
		m.addMsg("system", m.style("succ", " Session compacted — context summarized").Render())

	case "export":
		exp, err := m.app.session.Export()
		if err != nil {
			m.addMsg("system", m.style("err", "Export failed: "+err.Error()).Render())
			return nil
		}
		fname := fmt.Sprintf("codeagent-%s.md", time.Now().Format("2006-01-02-150405"))
		os.WriteFile(fname, []byte(exp), 0644)
		m.addMsg("system", m.style("succ", " Exported to "+fname).Render())

	case "share":
		url, err := m.app.session.Share()
		if err != nil {
			m.addMsg("system", m.style("err", "Share failed").Render())
			return nil
		}
		m.addMsg("system", m.style("succ", " Shared: "+url).Render())

	case "details":
		m.showDetails = !m.showDetails
		status := "ON"
		if !m.showDetails {
			status = "OFF"
		}
		m.addMsg("system", "Tool execution details: "+status)

	case "thinking":
		m.showThinking = !m.showThinking
		status := "ON"
		if !m.showThinking {
			status = "OFF"
		}
		m.addMsg("system", "Thinking blocks: "+status)

	case "themes":
		themeNames := make([]string, 0, len(Themes))
		for name := range Themes {
			themeNames = append(themeNames, name)
		}
		sort.Strings(themeNames)

		var sb strings.Builder
		sb.WriteString("Available Themes:\n\n")
		for _, name := range themeNames {
			t := Themes[name]
			current := ""
			if m.app.cfg.Theme == name {
				current = " ← current"
			}
			sb.WriteString(fmt.Sprintf("  %s — %s%s\n", m.style("prim", name).Render(), t.Name, m.style("dim", current).Render()))
		}
		sb.WriteString("\nConfigure in ~/.config/codeagent/codeagent.json")
		m.addMsg("system", sb.String())

	case "permission", "perm":
		m.addMsg("system", `Current Permission Rules:

  ✅ read       — Allowed
  ✅ grep       — Allowed
  ✅ glob       — Allowed
  ✅ webfetch   — Allowed
  ✅ question   — Allowed
  ⚠️  bash       — Ask before running
  ⚠️  edit       — Ask before modifying
  ⚠️  write      — Ask before creating
  ⚠️  websearch  — Ask
  ⚠️  task       — Ask
  ⚠️  todowrite  — Ask

  Configure in ~/.config/codeagent/codeagent.json`)

	case "editor":
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		m.addMsg("system", "Opening "+editor+"... (set $EDITOR to change)")

	case "palette":
		m.showPalette = !m.showPalette
		m.paletteIdx = 0

	default:
		m.addMsg("system", m.style("err", " Unknown command: /"+cmd).Render()+"\n"+m.style("dim", " Type /help for available commands").Render())
	}
	return nil
}

func (m *Model) renderHelp() string {
	return `Keyboard Shortcuts:

  Tab         Switch mode (build → plan → debug → review → docs)
  Ctrl+C/D    Exit CodeAgent
  Ctrl+P      Toggle command palette
  Ctrl+N      New session
  Ctrl+Z      Undo last change
  Ctrl+Y      Redo undone change
  Ctrl+L      Clear display
  ↑/↓         Scroll up/down
  PgUp/PgDown Page scroll

Slash Commands:

  /connect <provider> [key]   Add API key for a provider
  /model <provider/model>     Switch AI model
  /init                       Create AGENTS.md for your project
  /undo                       Revert last changes (git)
  /redo                       Restore undone changes
  /new                        Start a fresh session
  /sessions                   List saved sessions
  /compact                    Summarize & compact context
  /export                     Save conversation to Markdown
  /share                      Get a shareable session link
  /details                    Toggle tool execution details
  /thinking                   Toggle AI reasoning display
  /themes                     List available color themes
  /models                     List all available AI models
  /permission                 View tool access rules
  /palette                    Toggle command palette
  /editor                     Open external text editor
  /help                       Show this help screen
  /exit                       Quit CodeAgent

Tips:
  @filename     Attach file contents to your message
  !command      Run a shell command and see output
  Tab           Switch between agent modes
  Ctrl+P        Open command palette for quick access`
}

func doConnect(args []string, cfg *config.Config) string {
	if len(args) < 1 {
		return "Usage: /connect <provider> [api_key]"
	}
	provider := strings.ToLower(args[0])
	apiKey := ""
	if len(args) > 1 {
		apiKey = args[1]
	}

	supported := map[string]string{
		"openai": "OPENAI_API_KEY", "anthropic": "ANTHROPIC_API_KEY",
		"google": "GOOGLE_API_KEY", "groq": "GROQ_API_KEY",
		"openrouter": "OPENROUTER_API_KEY", "ollama": "",
		"mistral": "MISTRAL_API_KEY", "deepseek": "DEEPSEEK_API_KEY",
		"github-copilot": "GITHUB_TOKEN", "cohere": "COHERE_API_KEY",
		"perplexity": "PERPLEXITY_API_KEY", "xai": "XAI_API_KEY",
		"togetherai": "TOGETHERAI_API_KEY", "deepinfra": "DEEPINFRA_API_KEY",
		"cerebras": "CEREBRAS_API_KEY", "alibaba": "ALIBABA_API_KEY",
		"venice": "VENICE_API_KEY",
	}
	envKey, ok := supported[provider]
	if !ok && provider != "ollama" && provider != "azure" && provider != "aws-bedrock" {
		return "Unsupported provider: " + provider + "\nSupported: openai, anthropic, google, groq, openrouter, ollama, mistral, deepseek, github-copilot, cohere, perplexity, xai, togetherai, deepinfra, cerebras, alibaba, venice, azure, aws-bedrock"
	}

	if apiKey == "" && provider != "ollama" && provider != "azure" && provider != "aws-bedrock" {
		return provider + " requires an API key.\nUsage: /connect " + provider + " <your-key>\nOr set " + envKey + " environment variable."
	}

	creds, err := cfg.GetCredentialStore().LoadCredentials()
	if err != nil {
		creds = make(map[string]string)
	}
	creds[provider] = apiKey
	if err := cfg.GetCredentialStore().SaveCredentials(creds); err != nil {
		return " Failed to save: " + err.Error()
	}

	if apiKey != "" {
		shown := apiKey
		if len(shown) > 8 {
			shown = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
		}
		return " Connected to " + provider + "\nKey: " + shown
	}
	return " Connected to " + provider + " (local)"
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
