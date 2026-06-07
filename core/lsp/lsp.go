package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type LanguageServer struct {
	Name         string
	Command      string
	Args         []string
	LanguageID   string
	FilePatterns []string

	mu           sync.Mutex
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       *bufio.Scanner
	capabilities map[string]bool
	msgID        int
	initialized  bool
	buffer       strings.Builder
}

type LSPDiagnostic struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Severity int    `json:"severity"`
	Source   string `json:"source"`
	Code     string `json:"code,omitempty"`
}

type LSPResult struct {
	Server      string          `json:"server"`
	Diagnostics []LSPDiagnostic `json:"diagnostics"`
	Symbols     []LSPDocumentSymbol `json:"symbols,omitempty"`
	Error       string          `json:"error,omitempty"`
}

type LSPDocumentSymbol struct {
	Name           string `json:"name"`
	Kind           int    `json:"kind"`
	Detail         string `json:"detail"`
	RangeStartLine int    `json:"range_start_line"`
	RangeEndLine   int    `json:"range_end_line"`
}

type LanguageServerProtocol struct {
	servers []*LanguageServer
	mu      sync.Mutex
}

func NewLSP() *LanguageServerProtocol {
	return &LanguageServerProtocol{
		servers: make([]*LanguageServer, 0),
	}
}

func (l *LanguageServerProtocol) DetectAndStart(dir string) []*LanguageServer {
	l.mu.Lock()
	defer l.mu.Unlock()

	started := make([]*LanguageServer, 0)
	detectors := []struct {
		check        string
		name         string
		command      string
		args         []string
		languageID   string
		filePatterns []string
	}{
		{"go.mod", "gopls", "gopls", []string{"serve"}, "go", []string{"*.go"}},
		{"package.json", "typescript-language-server", "typescript-language-server", []string{"--stdio"}, "typescript", []string{"*.ts", "*.tsx", "*.js", "*.jsx"}},
		{"pyproject.toml", "pyright", "pyright", []string{"--stdio"}, "python", []string{"*.py"}},
		{"Cargo.toml", "rust-analyzer", "rust-analyzer", []string{}, "rust", []string{"*.rs"}},
		{"composer.json", "phpactor", "phpactor", []string{"language-server"}, "php", []string{"*.php"}},
		{"Gemfile", "solargraph", "solargraph", []string{"stdio"}, "ruby", []string{"*.rb"}},
		{"build.gradle", "gradle-language-server", "gradle-language-server", []string{}, "java", []string{"*.java", "*.gradle"}},
		{"CMakeLists.txt", "clangd", "clangd", []string{}, "cpp", []string{"*.cpp", "*.c", "*.h", "*.hpp"}},
		{"project.clj", "clojure-lsp", "clojure-lsp", []string{}, "clojure", []string{"*.clj", "*.cljs"}},
		{"mix.exs", "elixir-ls", "elixir-ls", []string{}, "elixir", []string{"*.ex", "*.exs"}},
	}

	for _, d := range detectors {
		checkPath := filepath.Join(dir, d.check)
		if _, err := os.Stat(checkPath); err != nil {
			continue
		}
		if _, err := exec.LookPath(d.command); err != nil {
			continue
		}

		server := &LanguageServer{
			Name:         d.name,
			Command:      d.command,
			Args:         d.args,
			LanguageID:   d.languageID,
			FilePatterns: d.filePatterns,
			capabilities: make(map[string]bool),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Start(ctx); err == nil {
			l.servers = append(l.servers, server)
			started = append(started, server)
		}
	}

	return started
}

func (l *LanguageServerProtocol) GetDiagnostics(file string) ([]LSPResult, error) {
	l.mu.Lock()
	servers := make([]*LanguageServer, len(l.servers))
	copy(servers, l.servers)
	l.mu.Unlock()

	var results []LSPResult
	for _, server := range servers {
		if !server.matchesFile(file) {
			continue
		}
		diags, err := server.GetDiagnostics(file)
		result := LSPResult{
			Server:      server.Name,
			Diagnostics: diags,
		}
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results, nil
}

func (l *LanguageServerProtocol) GetSymbols(file string) ([]LSPDocumentSymbol, error) {
	l.mu.Lock()
	servers := make([]*LanguageServer, len(l.servers))
	copy(servers, l.servers)
	l.mu.Unlock()

	var allSymbols []LSPDocumentSymbol
	for _, server := range servers {
		if !server.matchesFile(file) {
			continue
		}
		symbols, err := server.GetDocumentSymbols(file)
		if err == nil {
			allSymbols = append(allSymbols, symbols...)
		}
	}
	return allSymbols, nil
}

func (l *LanguageServerProtocol) GetHover(file string, line, col int) (string, error) {
	l.mu.Lock()
	servers := make([]*LanguageServer, len(l.servers))
	copy(servers, l.servers)
	l.mu.Unlock()

	for _, server := range servers {
		if !server.matchesFile(file) {
			continue
		}
		hover, err := server.GetHover(file, line, col)
		if err == nil && hover != "" {
			return hover, nil
		}
	}
	return "", nil
}

func (l *LanguageServerProtocol) GetCompletion(file string, line, col int) ([]string, error) {
	l.mu.Lock()
	servers := make([]*LanguageServer, len(l.servers))
	copy(servers, l.servers)
	l.mu.Unlock()

	for _, server := range servers {
		if !server.matchesFile(file) {
			continue
		}
		completions, err := server.GetCompletion(file, line, col)
		if err == nil && len(completions) > 0 {
			return completions, nil
		}
	}
	return nil, nil
}

func (l *LanguageServerProtocol) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, server := range l.servers {
		server.Stop()
	}
	l.servers = nil
}

func (ls *LanguageServer) matchesFile(file string) bool {
	ext := filepath.Ext(file)
	if ext == "" {
		return true
	}
	for _, pattern := range ls.FilePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(file)); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, "*"+ext); matched {
			return true
		}
	}
	return false
}

func (ls *LanguageServer) Start(ctx context.Context) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if ls.cmd != nil && ls.cmd.Process != nil {
		return nil
	}

	// Use exec.Command (not CommandContext) so the process outlives the startup timeout.
	// The context is only used to bound the initialization handshake.
	cmd := exec.Command(ls.Command, ls.Args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start LSP %s: %w", ls.Name, err)
	}

	ls.cmd = cmd
	ls.stdin = stdin
	ls.stdout = bufio.NewScanner(stdout)
	ls.stdout.Split(scanLSPMessages)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "error") || strings.Contains(line, "Error") {
				fmt.Fprintf(os.Stderr, "[LSP %s] %s\n", ls.Name, line)
			}
		}
	}()

	if err := ls.initialize(); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("LSP initialize failed: %w", err)
	}

	return nil
}

func (ls *LanguageServer) Stop() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if ls.cmd != nil && ls.cmd.Process != nil {
		ls.sendNotification("exit", nil)
		return ls.cmd.Process.Kill()
	}
	return nil
}

func (ls *LanguageServer) initialize() error {
	params := map[string]interface{}{
		"processId":             os.Getpid(),
		"clientInfo":            map[string]string{"name": "CodeAgent", "version": "1.0.0"},
		"capabilities":          map[string]interface{}{},
		"workspaceFolders":      []map[string]string{},
		"initializationOptions": nil,
	}

	resp, err := ls.sendRequest("initialize", params)
	if err != nil {
		return err
	}

	var result struct {
		Capabilities map[string]interface{} `json:"capabilities"`
	}
	if err := json.Unmarshal(resp, &result); err == nil {
		for k, v := range result.Capabilities {
			if boolVal, ok := v.(bool); ok {
				ls.capabilities[k] = boolVal
			}
		}
	}

	ls.sendNotification("initialized", map[string]interface{}{})
	ls.initialized = true
	return nil
}

func (ls *LanguageServer) isOpen(file string) bool {
	// LSP requires files to be opened via didOpen before diagnostics.
	// In a full implementation, track opened files in a map[string]bool.
	// For now, always return false so openFile is called before queries.
	return false
}

func (ls *LanguageServer) openFile(file string) error {
	uri := pathToURI(file)
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": ls.LanguageID,
			"version":    1,
			"text":       string(content),
		},
	}

	ls.sendNotification("textDocument/didOpen", params)
	return nil
}

func (ls *LanguageServer) GetDiagnostics(file string) ([]LSPDiagnostic, error) {
	ls.mu.Lock()
	if !ls.initialized {
		ls.mu.Unlock()
		return nil, nil
	}
	ls.mu.Unlock()

	uri := pathToURI(file)

	if !ls.isOpen(file) {
		ls.openFile(file)
	}

	publishParams := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
	}

	ls.sendNotification("textDocument/didChange", publishParams)

	time.Sleep(200 * time.Millisecond)

	resp, err := ls.sendRequest("textDocument/publishDiagnostics", publishParams)
	if err != nil {
		return nil, err
	}

	if len(resp) == 0 || string(resp) == "null" {
		return nil, nil
	}

	var diagResult struct {
		URI         string `json:"uri"`
		Diagnostics []struct {
			Range struct {
				Start struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"start"`
				End struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"end"`
			} `json:"range"`
			Severity int    `json:"severity"`
			Code     string `json:"code,omitempty"`
			Source   string `json:"source,omitempty"`
			Message  string `json:"message"`
		} `json:"diagnostics"`
	}

	if err := json.Unmarshal(resp, &diagResult); err != nil {
		return nil, err
	}

	diags := make([]LSPDiagnostic, len(diagResult.Diagnostics))
	for i, d := range diagResult.Diagnostics {
		diags[i] = LSPDiagnostic{
			File:     uriToPath(diagResult.URI),
			Line:     d.Range.Start.Line + 1,
			Column:   d.Range.Start.Character + 1,
			Message:  d.Message,
			Severity: d.Severity,
			Source:   d.Source,
			Code:     d.Code,
		}
	}

	return diags, nil
}

func (ls *LanguageServer) GetDocumentSymbols(file string) ([]LSPDocumentSymbol, error) {
	ls.mu.Lock()
	if !ls.initialized {
		ls.mu.Unlock()
		return nil, nil
	}
	ls.mu.Unlock()

	uri := pathToURI(file)

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
	}

	resp, err := ls.sendRequest("textDocument/documentSymbol", params)
	if err != nil {
		return nil, err
	}

	var symbols []struct {
		Name           string `json:"name"`
		Kind           int    `json:"kind"`
		Detail         string `json:"detail"`
		Range          struct {
			Start struct {
				Line      int `json:"line"`
				Character int `json:"character"`
			} `json:"start"`
			End struct {
				Line      int `json:"line"`
				Character int `json:"character"`
			} `json:"end"`
		} `json:"range"`
	}

	if err := json.Unmarshal(resp, &symbols); err != nil {
		return nil, err
	}

	result := make([]LSPDocumentSymbol, len(symbols))
	for i, s := range symbols {
		result[i] = LSPDocumentSymbol{
			Name:           s.Name,
			Kind:           s.Kind,
			Detail:         s.Detail,
			RangeStartLine: s.Range.Start.Line + 1,
			RangeEndLine:   s.Range.End.Line + 1,
		}
	}

	return result, nil
}

func (ls *LanguageServer) GetHover(file string, line, col int) (string, error) {
	ls.mu.Lock()
	if !ls.initialized {
		ls.mu.Unlock()
		return "", nil
	}
	ls.mu.Unlock()

	uri := pathToURI(file)
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
		"position": map[string]interface{}{
			"line":      line - 1,
			"character": col - 1,
		},
	}

	resp, err := ls.sendRequest("textDocument/hover", params)
	if err != nil {
		return "", err
	}

	var hoverResult struct {
		Contents map[string]interface{} `json:"contents"`
	}
	if err := json.Unmarshal(resp, &hoverResult); err != nil {
		return "", nil
	}

	if hoverResult.Contents == nil {
		return "", nil
	}

	if value, ok := hoverResult.Contents["value"]; ok {
		return fmt.Sprintf("%v", value), nil
	}
	if kind, ok := hoverResult.Contents["kind"]; ok {
		return fmt.Sprintf("%v", kind), nil
	}

	return "", nil
}

func (ls *LanguageServer) GetCompletion(file string, line, col int) ([]string, error) {
	ls.mu.Lock()
	if !ls.initialized {
		ls.mu.Unlock()
		return nil, nil
	}
	ls.mu.Unlock()

	uri := pathToURI(file)
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
		"position": map[string]interface{}{
			"line":      line - 1,
			"character": col - 1,
		},
		"context": map[string]interface{}{
			"triggerKind": 1,
		},
	}

	resp, err := ls.sendRequest("textDocument/completion", params)
	if err != nil {
		return nil, err
	}

	var completionResult struct {
		Items []struct {
			Label       string `json:"label"`
			Kind        int    `json:"kind"`
			Detail      string `json:"detail"`
			InsertText  string `json:"insertText"`
		} `json:"items"`
	}

	if err := json.Unmarshal(resp, &completionResult); err != nil {
		var items []struct {
			Label       string `json:"label"`
			Kind        int    `json:"kind"`
			Detail      string `json:"detail"`
			InsertText  string `json:"insertText"`
		}
		if err := json.Unmarshal(resp, &items); err != nil {
			return nil, nil
		}
		completionResult.Items = items
	}

	result := make([]string, len(completionResult.Items))
	for i, item := range completionResult.Items {
		text := item.InsertText
		if text == "" {
			text = item.Label
		}
		result[i] = text
	}

	return result, nil
}

type lspMessage struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (ls *LanguageServer) sendRequest(method string, params interface{}) (json.RawMessage, error) {
	ls.msgID++
	id := ls.msgID

	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}

	if err := ls.writeMessage(msg); err != nil {
		return nil, err
	}

	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			return nil, fmt.Errorf("LSP request %s timed out", method)
		default:
			if ls.stdout.Scan() {
				line := ls.stdout.Text()
				var response lspMessage
				if err := json.Unmarshal([]byte(line), &response); err != nil {
					continue
				}
				if response.ID != nil && *response.ID == id {
					if response.Error != nil {
						return nil, fmt.Errorf("LSP error: %s", response.Error.Message)
					}
					return response.Result, nil
				}
				if response.Method != "" {
					ls.handleServerNotification(response.Method, response.Params)
				}
			}
		}
	}
}

func (ls *LanguageServer) sendNotification(method string, params interface{}) error {
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	return ls.writeMessage(msg)
}

func (ls *LanguageServer) writeMessage(msg interface{}) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := ls.stdin.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := ls.stdin.Write(data); err != nil {
		return err
	}

	return nil
}

func (ls *LanguageServer) handleServerNotification(method string, params json.RawMessage) {
	switch method {
	case "textDocument/publishDiagnostics":
		var diagParams struct {
			URI         string `json:"uri"`
			Diagnostics []struct {
				Range struct {
					Start struct {
						Line      int `json:"line"`
						Character int `json:"character"`
					} `json:"start"`
				} `json:"range"`
				Severity int    `json:"severity"`
				Message  string `json:"message"`
			} `json:"diagnostics"`
		}
		if err := json.Unmarshal(params, &diagParams); err == nil {
			file := uriToPath(diagParams.URI)
			for _, d := range diagParams.Diagnostics {
				fmt.Fprintf(os.Stderr, "[LSP Diag] %s:%d: %s\n", file, d.Range.Start.Line+1, d.Message)
			}
		}
	case "window/showMessage":
		var msgParams struct {
			Type    int    `json:"type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &msgParams); err == nil {
			fmt.Fprintf(os.Stderr, "[LSP %s] %s\n", ls.Name, msgParams.Message)
		}
	}
}

func scanLSPMessages(data []byte, atEOF bool) (advance int, token []byte, err error) {
	headerEnd := strings.Index(string(data), "\r\n\r\n")
	if headerEnd == -1 {
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}

	header := string(data[:headerEnd])
	var contentLength int
	fmt.Sscanf(header, "Content-Length: %d", &contentLength)

	bodyStart := headerEnd + 4
	if len(data) < bodyStart+contentLength {
		return 0, nil, nil
	}

	return bodyStart + contentLength, data[bodyStart : bodyStart+contentLength], nil
}

func pathToURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	abs = filepath.ToSlash(abs)
	if !strings.HasPrefix(abs, "/") {
		abs = "/" + abs
	}
	return "file://" + abs
}

func uriToPath(uri string) string {
	uri = strings.TrimPrefix(uri, "file://")
	if strings.HasPrefix(uri, "/") {
		uri = uri[1:]
	}
	uri = filepath.FromSlash(uri)
	return uri
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func cmdExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
