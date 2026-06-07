package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"codeagent/core/agent"
	"codeagent/core/bus"
	"codeagent/config"
	"codeagent/providers"
	"codeagent/core/session"
)

type Options struct {
	Port        int
	Hostname    string
	EnableMdns  bool
	MdnsDomain  string
	CorsOrigins []string
	EnableWeb   bool
}

type Server struct {
	cfg     *config.Config
	opts    Options
	httpSrv *http.Server
	store   *session.Store
	sseClients map[string]chan bus.Event
	sseMu      sync.RWMutex
}

func New(cfg *config.Config, opts Options) (*Server, error) {
	if opts.Port == 0 {
		opts.Port = 4096
	}
	if opts.Hostname == "" {
		opts.Hostname = "localhost"
	}

	store, err := session.NewStore(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}

	s := &Server{
		cfg:        cfg,
		opts:       opts,
		store:      store,
		sseClients: make(map[string]chan bus.Event),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("/v1/sessions", s.handleSessions)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/events", s.handleSSE)
	mux.HandleFunc("/v1/permissions", s.handlePermissions)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleRoot)

	if opts.EnableWeb {
		mux.HandleFunc("/web", s.handleWeb)
	}

	s.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", opts.Hostname, opts.Port),
		Handler: s.withCORS(s.withAuth(mux)),
	}

	// Subscribe to bus events for SSE
	bus.Subscribe(bus.EventUserInput, s.handleBusEvent)
	bus.Subscribe(bus.EventAssistantResp, s.handleBusEvent)
	bus.Subscribe(bus.EventToolCall, s.handleBusEvent)
	bus.Subscribe(bus.EventToolResult, s.handleBusEvent)
	bus.Subscribe(bus.EventModeChange, s.handleBusEvent)
	bus.Subscribe(bus.EventError, s.handleBusEvent)

	return s, nil
}

func (s *Server) handleBusEvent(event bus.Event) {
	s.sseMu.RLock()
	defer s.sseMu.RUnlock()

	data, _ := json.Marshal(event)
	for _, client := range s.sseClients {
		select {
		case client <- event:
		default:
		}
	}
	fmt.Fprintf(os.Stderr, "[SSE] Event: %s\n", event.Type)
	if len(data) > 0 {
		fmt.Fprintf(os.Stderr, "[SSE] Data: %s\n", string(data))
	}
}

func (s *Server) Start() error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpSrv.Shutdown(ctx)
	}()

	fmt.Fprintf(os.Stderr, "CodeAgent server listening on %s\n", s.httpSrv.Addr)
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	password := os.Getenv("CODEAGENT_SERVER_PASSWORD")
	username := os.Getenv("CODEAGENT_SERVER_USERNAME")
	if username == "" {
		username = "codeagent"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/events" && password == "" {
			next.ServeHTTP(w, r)
			return
		}
		if password != "" {
			u, p, ok := r.BasicAuth()
			if !ok || u != username || p != password {
				w.Header().Set("WWW-Authenticate", `Basic realm="CodeAgent"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if s.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range s.opts.CorsOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": "1.0.0",
		"server":  "codeagent",
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":        "CodeAgent API",
		"version":     "1.0.0",
		"endpoints":   []string{"/v1/chat/completions", "/v1/sessions", "/v1/models", "/v1/events", "/v1/permissions", "/health"},
		"description": "Open source AI coding agent API",
	})
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	eventCh := make(chan bus.Event, 100)

	s.sseMu.Lock()
	s.sseClients[clientID] = eventCh
	s.sseMu.Unlock()

	defer func() {
		s.sseMu.Lock()
		delete(s.sseClients, clientID)
		s.sseMu.Unlock()
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()
		}
	}
}

func (s *Server) handlePermissions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.cfg.Permissions)
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model    string          `json:"model"`
		Messages []providers.Message   `json:"messages"`
		Stream   bool            `json:"stream"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sess := session.New(s.cfg, s.store)
	sess.Model = req.Model
	for _, m := range req.Messages {
		sess.AddMessage(m.Role, m.Content)
	}

	provider, err := providers.GetProvider(s.cfg, req.Model)
	if err != nil {
		http.Error(w, fmt.Sprintf("Provider error: %v", err), http.StatusBadRequest)
		return
	}

	ag, err := agent.New(s.cfg, sess, provider)
	if err != nil {
		http.Error(w, fmt.Sprintf("Agent error: %v", err), http.StatusInternalServerError)
		return
	}

	if req.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		stream, err := ag.RunStream(req.Messages[len(req.Messages)-1].Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var fullResponse strings.Builder
		for event := range stream {
			switch event.Type {
			case providers.StreamEventText:
				fullResponse.WriteString(event.Content)
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	lastMsg := req.Messages[len(req.Messages)-1].Content
	resp, err := ag.Run(lastMsg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		sessions, err := s.store.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	case "DELETE":
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id parameter required", http.StatusBadRequest)
			return
		}
		if err := s.store.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	registry := providers.NewRegistry()
	models := registry.ListModels(s.cfg.Providers)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, webHTML)
}

var webHTML = `<!DOCTYPE html>
<html>
<head>
	<title>CodeAgent Web</title>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #1a1b26; color: #c0caf5; height: 100vh; display: flex; flex-direction: column; }
		.header { padding: 16px 20px; border-bottom: 1px solid #3b4261; background: #1e1e2e; }
		.header h1 { color: #7aa2f7; font-size: 20px; display: inline; }
		.header .mode { color: #a6e3a1; margin-left: 12px; font-size: 12px; background: #2a3b5c; padding: 4px 8px; border-radius: 4px; }
		.container { max-width: 900px; margin: 0 auto; width: 100%; display: flex; flex-direction: column; height: 100vh; }
		#messages { flex: 1; overflow-y: auto; padding: 16px 20px; }
		.message { margin-bottom: 16px; padding: 12px 16px; border-radius: 8px; }
		.message.user { background: #2a3b5c; border-left: 3px solid #7aa2f7; }
		.message.assistant { background: #1f2a40; border-left: 3px solid #a6e3a1; }
		.message.system { background: transparent; border: none; color: #565f89; font-size: 13px; text-align: center; }
		.message .role { font-weight: bold; color: #7aa2f7; margin-bottom: 6px; font-size: 13px; }
		.message.assistant .role { color: #a6e3a1; }
		.message .content { line-height: 1.6; font-size: 14px; white-space: pre-wrap; }
		.message .content code { background: #1a1b26; padding: 2px 6px; border-radius: 4px; font-size: 13px; }
		.message .content pre { background: #1a1b26; padding: 12px; border-radius: 6px; overflow-x: auto; margin: 8px 0; }
		#input-area { padding: 12px 20px; border-top: 1px solid #3b4261; background: #1e1e2e; display: flex; gap: 8px; align-items: center; }
		#mode-btn { padding: 8px 14px; background: #2a3b5c; color: #a6e3a1; border: 1px solid #3b4261; border-radius: 6px; cursor: pointer; font-size: 13px; font-weight: bold; }
		#mode-btn:hover { background: #3b4261; }
		#input { flex: 1; padding: 12px 16px; border: 1px solid #3b4261; border-radius: 8px; background: #24283b; color: #c0caf5; font-size: 14px; outline: none; }
		#input:focus { border-color: #7aa2f7; }
		#send { padding: 12px 24px; background: #7aa2f7; color: #1a1b26; border: none; border-radius: 8px; cursor: pointer; font-weight: bold; font-size: 14px; }
		#send:hover { background: #89b4fa; }
		#send:disabled { opacity: 0.5; cursor: not-allowed; }
		.status { color: #565f89; font-size: 12px; padding: 4px 20px; background: #1e1e2e; }
		.tool-call { color: #565f89; font-size: 12px; margin: 4px 0; padding: 4px 8px; background: #1a1b26; border-radius: 4px; }
		@media (max-width: 600px) {
			#input-area { flex-wrap: wrap; }
			#send { width: 100%; }
		}
	</style>
</head>
<body>
	<div class="header">
		<div class="container">
			<h1>CodeAgent</h1>
			<span class="mode" id="mode-display">◆ BUILD</span>
		</div>
	</div>
	<div class="container">
		<div id="messages"></div>
		<div class="status" id="status">Ready <span id="model-status">no model configured</span></div>
		<div id="input-area">
			<button id="mode-btn" onclick="cycleMode()">◆ BUILD</button>
			<input type="text" id="input" placeholder="Ask CodeAgent to do anything... (@file, /help)" autofocus />
			<button id="send" onclick="send()">Send</button>
		</div>
	</div>
	<script>
		const modes = ['build', 'plan', 'debug', 'review', 'docs'];
		let currentMode = 0;
		const messagesEl = document.getElementById('messages');
		const inputEl = document.getElementById('input');
		const sendEl = document.getElementById('send');
		const statusEl = document.getElementById('status');
		const modeDisplay = document.getElementById('mode-display');
		const modeBtn = document.getElementById('mode-btn');

		function addMessage(role, content, toolCalls) {
			const div = document.createElement('div');
			div.className = 'message ' + role;
			const roleDiv = document.createElement('div');
			roleDiv.className = 'role';
			const roleNames = { user: 'You', assistant: 'CodeAgent', system: '' };
			roleDiv.textContent = roleNames[role] || role;
			const contentDiv = document.createElement('div');
			contentDiv.className = 'content';
			contentDiv.innerHTML = renderMarkdown(content);
			div.appendChild(roleDiv);
			if (toolCalls && toolCalls.length > 0) {
				toolCalls.forEach(tc => {
					const tcDiv = document.createElement('div');
					tcDiv.className = 'tool-call';
					tcDiv.textContent = tc.tool + ' - completed';
					div.appendChild(tcDiv);
				});
			}
			div.appendChild(contentDiv);
			messagesEl.appendChild(div);
			messagesEl.scrollTop = messagesEl.scrollHeight;
		}

		function renderMarkdown(text) {
			var BT = '\x60';
			var r = text
				.replace(/&/g, '&amp;')
				.replace(/</g, '&lt;')
				.replace(/>/g, '&gt;')
				.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
				.replace(/\*(.+?)\*/g, '<em>$1</em>')
				.replace(/\n/g, '<br>');
			r = r.replace(new RegExp(BT + BT + BT + '([\\s\\S]*?)' + BT + BT + BT, 'g'), '<pre><code>$1</code></pre>');
			r = r.replace(new RegExp('(?<!' + BT + ')' + BT + '([^' + BT + ']+)' + BT, 'g'), '<code>$1</code>');
			return r;
		}

		function cycleMode() {
			currentMode = (currentMode + 1) % modes.length;
			const mode = modes[currentMode];
			const labels = { build: 'BUILD', plan: 'PLAN', debug: 'DEBUG', review: 'REVIEW', docs: 'DOCS' };
			const colors = { build: '#a6e3a1', plan: '#f9e2af', debug: '#f38ba8', review: '#89b4fa', docs: '#cba6f7' };
			modeBtn.textContent = labels[mode];
			modeBtn.style.color = colors[mode];
			modeDisplay.textContent = labels[mode];
			modeDisplay.style.color = colors[mode];
		}

		async function send() {
			const text = inputEl.value.trim();
			if (!text) return;

			addMessage('user', text);
			inputEl.value = '';
			sendEl.disabled = true;
			statusEl.innerHTML = 'Thinking...';

			try {
				const resp = await fetch('/v1/chat/completions', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						model: 'openai/gpt-5',
						messages: [{ role: 'user', content: text }],
						stream: false
					})
				});
				if (!resp.ok) {
					const err = await resp.text();
					throw new Error(err);
				}
				const data = await resp.json();
				addMessage('assistant', data.text || data.content, (data.toolCalls || []).map(tc => ({
					tool: tc.function ? tc.function.name : 'unknown',
					status: 'completed'
				})));
				statusEl.innerHTML = 'Ready';
			} catch (err) {
				addMessage('system', 'Error: ' + err.message);
				statusEl.innerHTML = 'Error';
			} finally {
				sendEl.disabled = false;
				inputEl.focus();
			}
		}

		inputEl.addEventListener('keydown', (e) => { if (e.key === 'Enter') send(); });

		async function initSSE() {
			const evtSource = new EventSource('/v1/events');
			evtSource.onmessage = (e) => {
				try {
					const event = JSON.parse(e.data);
				} catch (err) {}
			};
			evtSource.onerror = () => {};
		}
		initSSE();
	</script>
</body>
</html>`
