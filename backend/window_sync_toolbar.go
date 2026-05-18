package backend

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	windowSyncToolbarArg       = "--window-sync-toolbar"
	windowSyncToolbarPortArg   = "--window-sync-toolbar-port"
	windowSyncToolbarTokenArg  = "--window-sync-toolbar-token"
	windowSyncToolbarWidth     = 520
	windowSyncToolbarHeight    = 76
	windowSyncToolbarTopOffset = 18
)

type windowSyncToolbarController struct {
	mu      sync.Mutex
	app     *App
	server  *http.Server
	port    int
	token   string
	process *exec.Cmd
}

type windowSyncToolbarCommandInput struct {
	Command string                              `json:"command"`
	Mode    string                              `json:"mode"`
	Text    string                              `json:"text"`
	Items   []WindowSyncBatchInputDifferentItem `json:"items"`
}

type WindowSyncToolbarConfig struct {
	Port   int    `json:"port"`
	Token  string `json:"token"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

func (c *windowSyncToolbarController) Show(app *App, state *WindowSyncState) error {
	if app == nil || state == nil || !state.Active {
		return nil
	}
	if err := c.ensureServer(app); err != nil {
		return err
	}
	return c.ensureProcess(app)
}

func (c *windowSyncToolbarController) Update(state *WindowSyncState) error {
	if state == nil || !state.Active {
		return nil
	}
	return c.ensureProcess(c.app)
}

func (c *windowSyncToolbarController) Hide() error {
	c.mu.Lock()
	cmd := c.process
	server := c.server
	c.process = nil
	c.server = nil
	c.app = nil
	c.port = 0
	c.token = ""
	c.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = server.Shutdown(ctx)
		cancel()
	}
	return nil
}

func (c *windowSyncToolbarController) ensureServer(app *App) error {
	c.mu.Lock()
	if c.server != nil {
		c.app = app
		c.mu.Unlock()
		return nil
	}
	token, err := randomWindowSyncToolbarToken()
	if err != nil {
		c.mu.Unlock()
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("启动窗口同步工具栏服务失败: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}
	c.app = app
	c.server = server
	c.port = port
	c.token = token
	mux.HandleFunc("/state", c.handleState)
	mux.HandleFunc("/command", c.handleCommand)
	c.mu.Unlock()

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("窗口同步工具栏服务异常退出: %v\n", err)
		}
	}()
	return nil
}

func (c *windowSyncToolbarController) ensureProcess(app *App) error {
	c.mu.Lock()
	if c.process != nil && c.process.Process != nil && c.process.ProcessState == nil {
		c.mu.Unlock()
		return nil
	}
	port := c.port
	token := c.token
	c.mu.Unlock()

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取当前程序路径失败: %w", err)
	}
	config := windowSyncToolbarInitialConfig(port, token)
	args := []string{
		windowSyncToolbarArg,
		fmt.Sprintf("%s=%d", windowSyncToolbarPortArg, port),
		fmt.Sprintf("%s=%s", windowSyncToolbarTokenArg, token),
	}
	cmd := exec.Command(exePath, args...)
	cmd.Env = append(os.Environ(),
		"TRACE_WINDOW_SYNC_TOOLBAR=1",
		fmt.Sprintf("TRACE_WINDOW_SYNC_TOOLBAR_PORT=%d", port),
		fmt.Sprintf("TRACE_WINDOW_SYNC_TOOLBAR_TOKEN=%s", token),
		fmt.Sprintf("TRACE_WINDOW_SYNC_TOOLBAR_CONFIG=%s", config),
	)
	if app != nil {
		if root := strings.TrimSpace(app.appRootAbs()); root != "" {
			cmd.Dir = root
		}
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动窗口同步悬浮工具栏失败: %w", err)
	}
	go func() {
		_ = cmd.Wait()
		c.mu.Lock()
		if c.process == cmd {
			c.process = nil
		}
		c.mu.Unlock()
	}()

	c.mu.Lock()
	c.process = cmd
	c.mu.Unlock()
	return nil
}

func (c *windowSyncToolbarController) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		writeWindowSyncToolbarCORS(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !c.authorize(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeWindowSyncToolbarCORS(w)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	c.mu.Lock()
	app := c.app
	c.mu.Unlock()
	var state *WindowSyncState
	if app != nil {
		state = app.WindowSyncGetState()
	}
	writeWindowSyncToolbarJSON(w, state)
}

func (c *windowSyncToolbarController) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		writeWindowSyncToolbarCORS(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !c.authorize(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeWindowSyncToolbarCORS(w)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input windowSyncToolbarCommandInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeWindowSyncToolbarCORS(w)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	c.mu.Lock()
	app := c.app
	c.mu.Unlock()
	if app == nil {
		writeWindowSyncToolbarCORS(w)
		http.Error(w, "window sync is not available", http.StatusGone)
		return
	}

	var (
		data any
		err  error
	)
	switch strings.TrimSpace(strings.ToLower(input.Command)) {
	case "show-all":
		data, err = app.WindowSyncShowAll()
	case "layout":
		data, err = app.WindowSyncApplyLayout(WindowSyncLayoutSettings{Mode: strings.TrimSpace(strings.ToLower(input.Mode))})
	case "pause":
		data, err = app.WindowSyncPause()
	case "resume":
		data, err = app.WindowSyncResume()
	case "stop":
		data, err = app.WindowSyncStop()
	case "batch-input-same":
		data, err = app.WindowSyncBatchInputSame(WindowSyncBatchInputSameInput{Text: input.Text})
	case "batch-input-different":
		data, err = app.WindowSyncBatchInputDifferent(WindowSyncBatchInputDifferentInput{Items: input.Items})
	default:
		writeWindowSyncToolbarCORS(w)
		http.Error(w, "unknown command", http.StatusBadRequest)
		return
	}
	if err != nil {
		writeWindowSyncToolbarCORS(w)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeWindowSyncToolbarJSON(w, data)
}

func (c *windowSyncToolbarController) authorize(w http.ResponseWriter, r *http.Request) bool {
	c.mu.Lock()
	token := c.token
	c.mu.Unlock()
	if token == "" || r.Header.Get("X-Trace-Toolbar-Token") != token {
		writeWindowSyncToolbarCORS(w)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	writeWindowSyncToolbarCORS(w)
	return true
}

func writeWindowSyncToolbarCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Trace-Toolbar-Token")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

func writeWindowSyncToolbarJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(data)
}

func randomWindowSyncToolbarToken() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("生成窗口同步工具栏令牌失败: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

func windowSyncToolbarInitialConfig(port int, token string) string {
	area := primaryWorkArea()
	xOffset := (area.Width - windowSyncToolbarWidth) / 2
	if xOffset < 0 {
		xOffset = 0
	}
	x := area.Left + xOffset
	y := area.Top + windowSyncToolbarTopOffset
	cfg := WindowSyncToolbarConfig{
		Port:   port,
		Token:  token,
		Width:  windowSyncToolbarWidth,
		Height: windowSyncToolbarHeight,
		X:      x,
		Y:      y,
	}
	data, _ := json.Marshal(cfg)
	return string(data)
}

func ParseWindowSyncToolbarArgs(args []string) WindowSyncToolbarConfig {
	cfg := WindowSyncToolbarConfig{
		Width:  windowSyncToolbarWidth,
		Height: windowSyncToolbarHeight,
	}
	if raw := strings.TrimSpace(os.Getenv("TRACE_WINDOW_SYNC_TOOLBAR_CONFIG")); raw != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	if cfg.Port <= 0 {
		cfg.Port = parseWindowSyncToolbarIntArg(args, windowSyncToolbarPortArg, os.Getenv("TRACE_WINDOW_SYNC_TOOLBAR_PORT"))
	}
	if strings.TrimSpace(cfg.Token) == "" {
		cfg.Token = parseWindowSyncToolbarStringArg(args, windowSyncToolbarTokenArg, os.Getenv("TRACE_WINDOW_SYNC_TOOLBAR_TOKEN"))
	}
	if cfg.Width <= 0 {
		cfg.Width = windowSyncToolbarWidth
	}
	if cfg.Height <= 0 {
		cfg.Height = windowSyncToolbarHeight
	}
	return cfg
}

func IsWindowSyncToolbarProcess(args []string) bool {
	if strings.TrimSpace(os.Getenv("TRACE_WINDOW_SYNC_TOOLBAR")) == "1" {
		return true
	}
	for _, arg := range args {
		if strings.TrimSpace(arg) == windowSyncToolbarArg {
			return true
		}
	}
	return false
}

func WindowSyncToolbarAssetMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL != nil && r.URL.Path == "/" {
			clone := new(http.Request)
			*clone = *r
			cloneURL := *r.URL
			clone.URL = &cloneURL
			values := cloneURL.Query()
			values.Set("toolbar", "1")
			cloneURL.Path = "/toolbar.html"
			cloneURL.RawQuery = values.Encode()
			next.ServeHTTP(w, clone)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func WindowSyncToolbarHTMLHandler(appRoot string, assets fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL == nil || r.URL.Path != "/toolbar.html" {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(assets, "frontend/dist/index.html")
		if err != nil {
			data, err = os.ReadFile(filepath.Join(appRoot, "frontend", "dist", "index.html"))
			if err != nil {
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data = bytes.ReplaceAll(data, []byte("/src/main.tsx"), []byte("/src/main.tsx?toolbar=1"))
		data = injectWindowSyncToolbarConfig(data, r.URL.Query().Get("toolbar") == "1")
		_, _ = w.Write(data)
	})
}

func injectWindowSyncToolbarConfig(data []byte, toolbar bool) []byte {
	if !toolbar {
		return data
	}
	config := strings.TrimSpace(os.Getenv("TRACE_WINDOW_SYNC_TOOLBAR_CONFIG"))
	if config == "" {
		return data
	}
	script := []byte("<script>window.__TRACE_WINDOW_SYNC_TOOLBAR_CONFIG__=JSON.parse(" + strconv.Quote(config) + ");history.replaceState(null,'','/?toolbar=1');</script>")
	if bytes.Contains(data, []byte("</head>")) {
		return bytes.Replace(data, []byte("</head>"), append(script, []byte("</head>")...), 1)
	}
	return append(script, data...)
}

func parseWindowSyncToolbarIntArg(args []string, name string, fallback string) int {
	value := parseWindowSyncToolbarStringArg(args, name, fallback)
	n, _ := strconv.Atoi(value)
	return n
}

func parseWindowSyncToolbarStringArg(args []string, name string, fallback string) string {
	prefix := name + "="
	for i, arg := range args {
		arg = strings.TrimSpace(arg)
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(arg, prefix))
		}
		if arg == name && i+1 < len(args) {
			return strings.TrimSpace(args[i+1])
		}
	}
	return strings.TrimSpace(fallback)
}
