package main

import (
	"ant-chrome/backend"
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"gopkg.in/yaml.v3"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/config.yml
var buildConfigYAML []byte

//go:embed build/appicon.png
var appIcon []byte

// appBuildVersion is injected by release builds with:
// go build -ldflags "-X main.appBuildVersion=0.0.87"
var appBuildVersion string

var appRoot string
var isDevMode bool

type App struct {
	*backend.App
}

type wailsBuildConfig struct {
	Info struct {
		Version string `yaml:"version"`
	} `yaml:"info"`
}

func normalizeStartupWindowSize(width int, height int) (int, int) {
	const (
		defaultWidth  = 1180
		defaultHeight = 720
	)
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}
	return width, height
}

func clampStartupWindowSize(width int, height int, minWidth int, minHeight int) (int, int) {
	const (
		fallbackMinWidth  = 900
		fallbackMinHeight = 560
		maxWidth          = 1360
		maxHeight         = 860
	)
	if minWidth <= 0 {
		minWidth = fallbackMinWidth
	}
	if minHeight <= 0 {
		minHeight = fallbackMinHeight
	}
	if width < minWidth {
		width = minWidth
	}
	if height < minHeight {
		height = minHeight
	}
	if width > maxWidth {
		width = maxWidth
	}
	if height > maxHeight {
		height = maxHeight
	}
	return width, height
}

func envFlagEnabled(name string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func normalizeBuildVersion(value string) string {
	version := strings.TrimSpace(value)
	if version == "" || strings.EqualFold(version, "unknown") {
		return ""
	}
	if len(version) > 1 && (version[0] == 'v' || version[0] == 'V') {
		version = strings.TrimSpace(version[1:])
	}
	return version
}

func resolveBuildVersion() string {
	if version := normalizeBuildVersion(appBuildVersion); version != "" {
		return version
	}
	if version := normalizeBuildVersion(os.Getenv("TRACE_BROWSER_VERSION")); version != "" {
		return version
	}

	var cfg wailsBuildConfig
	if err := yaml.Unmarshal(buildConfigYAML, &cfg); err != nil {
		log.Printf("解析 build/config.yml 版本信息失败: %v", err)
		return "dev"
	}

	version := normalizeBuildVersion(cfg.Info.Version)
	if version == "" {
		log.Printf("build/config.yml 未配置 info.version，回退为 dev")
		return "dev"
	}

	return version
}

func NewApp(appRoot, version string) *App {
	return &App{App: backend.NewApp(appRoot, version)}
}

const (
	wails3WindowSyncToolbarName           = "window-sync-toolbar"
	wails3WindowSyncToolbarExpandedWidth  = 780
	wails3WindowSyncToolbarExpandedHeight = 430
)

type wails3WindowSyncToolbarAdapter struct {
	mu                  sync.Mutex
	wailsApp            *application.App
	protoIPC            *backend.Wails3ProtoIPC
	window              application.Window
	width               int
	height              int
	startupDebugEnabled bool
}

func (a *wails3WindowSyncToolbarAdapter) Show(_ *backend.App, state *backend.WindowSyncState) error {
	if a == nil || state == nil || !state.Active {
		return nil
	}
	if a.protoIPC == nil {
		log.Printf("Wails3 Proto IPC 未初始化，窗口同步工具栏不会回退到旧 HTTP/子进程通道")
		return fmt.Errorf("wails3 proto ipc is not initialized")
	}
	cfg := backend.DefaultWindowSyncToolbarConfig()
	a.resetSize(cfg.Width, cfg.Height)
	return a.showWithConfig(cfg)
}

func (a *wails3WindowSyncToolbarAdapter) Update(state *backend.WindowSyncState) error {
	if a == nil || state == nil || !state.Active {
		return a.Hide()
	}
	if a.protoIPC == nil {
		return fmt.Errorf("wails3 proto ipc is not initialized")
	}
	return a.showWithConfig(backend.DefaultWindowSyncToolbarConfig())
}

func (a *wails3WindowSyncToolbarAdapter) Hide() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	window := a.window
	a.mu.Unlock()
	if window != nil {
		window.Close()
	}
	return nil
}

func (a *wails3WindowSyncToolbarAdapter) SetSize(width int, height int) error {
	if a == nil {
		return nil
	}
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid toolbar size")
	}
	width = clampInt(width, 320, 900)
	height = clampInt(height, 64, 560)

	a.mu.Lock()
	a.width = width
	a.height = height
	window := a.window
	a.mu.Unlock()

	if window != nil {
		window.SetSize(width, height)
		window.SetAlwaysOnTop(true)
	}
	return nil
}

func (a *wails3WindowSyncToolbarAdapter) resetSize(width int, height int) {
	a.mu.Lock()
	a.width = width
	a.height = height
	a.mu.Unlock()
}

func (a *wails3WindowSyncToolbarAdapter) showWithConfig(cfg backend.WindowSyncToolbarConfig) error {
	if a.wailsApp == nil {
		return fmt.Errorf("wails3 application is not initialized")
	}
	window := a.ensureWindow(cfg)
	if window == nil {
		return fmt.Errorf("create wails3 toolbar window failed")
	}
	a.mu.Lock()
	width := a.width
	height := a.height
	a.mu.Unlock()
	window.SetSize(width, height)
	window.SetAlwaysOnTop(true)
	window.Show()
	a.injectProtoConfig(window)
	return nil
}

func (a *wails3WindowSyncToolbarAdapter) ensureWindow(cfg backend.WindowSyncToolbarConfig) application.Window {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.width <= 0 {
		a.width = cfg.Width
	}
	if a.height <= 0 {
		a.height = cfg.Height
	}
	if a.window != nil {
		return a.window
	}

	toolbarWindow := a.wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             wails3WindowSyncToolbarName,
		Title:            "窗口同步",
		Width:            a.width,
		Height:           a.height,
		MinWidth:         cfg.Width,
		MinHeight:        cfg.Height,
		MaxWidth:         wails3WindowSyncToolbarExpandedWidth,
		MaxHeight:        wails3WindowSyncToolbarExpandedHeight,
		DisableResize:    true,
		Frameless:        true,
		AlwaysOnTop:      true,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(245, 247, 250, 0),
		InitialPosition:  application.WindowXY,
		X:                cfg.X,
		Y:                cfg.Y,
		URL:              "/?toolbar=1",
		JS:               a.protoIPC.ConfigScript(),
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: true,
			HiddenOnTaskbar:                   true,
		},
		MinimiseButtonState:   application.ButtonHidden,
		MaximiseButtonState:   application.ButtonHidden,
		CloseButtonState:      application.ButtonHidden,
		FullscreenButtonState: application.ButtonHidden,
	})
	toolbarWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		a.mu.Lock()
		if a.window == toolbarWindow {
			a.window = nil
		}
		a.mu.Unlock()
	})
	a.window = toolbarWindow
	if a.startupDebugEnabled {
		log.Printf("Wails3 窗口同步工具栏已创建为同进程多窗口: %s", wails3WindowSyncToolbarName)
	}
	return toolbarWindow
}

func (a *wails3WindowSyncToolbarAdapter) injectProtoConfig(window application.Window) {
	if a == nil || a.protoIPC == nil || window == nil {
		return
	}
	a.protoIPC.InjectConfig(window)
	go func() {
		for _, delay := range []time.Duration{250 * time.Millisecond, 900 * time.Millisecond} {
			time.Sleep(delay)
			a.protoIPC.InjectConfig(window)
		}
	}()
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func resolveAppRoot() {
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		tempDir := os.TempDir()
		if resolved, err := filepath.EvalSymlinks(exeDir); err == nil {
			exeDir = resolved
		}
		if resolved, err := filepath.EvalSymlinks(tempDir); err == nil {
			tempDir = resolved
		}

		exeDirLower := strings.ToLower(exeDir)
		inTemp := strings.HasPrefix(exeDirLower, strings.ToLower(tempDir))
		inBuildBin := strings.HasSuffix(filepath.ToSlash(exeDirLower), "/build/bin")
		if inTemp || inBuildBin {
			isDevMode = true
			if cwd, err := os.Getwd(); err == nil {
				appRoot = cwd
			} else {
				appRoot = "."
			}
			return
		}

		isDevMode = false
		appRoot = exeDir
		_ = os.Chdir(exeDir)
		return
	}

	if cwd, err := os.Getwd(); err == nil {
		appRoot = cwd
	} else {
		appRoot = "."
	}
}

func main() {
	resolveAppRoot()

	startupDebugEnabled := envFlagEnabled("ANT_BROWSER_DEBUG_STARTUP")
	if startupDebugEnabled {
		log.Printf("应用根目录: %s (dev=%v, wails3=true)", appRoot, isDevMode)
	}
	if err := backend.EnsureRuntimeLayout(appRoot); err != nil {
		log.Printf("准备用户数据目录失败: %v", err)
	}
	if startupDebugEnabled && backend.RuntimeUsesDetachedState(appRoot) {
		log.Printf("检测到安装目录需要只读运行，状态目录切换到: %s", backend.RuntimeStateRoot(appRoot))
	}

	buildVersion := resolveBuildVersion()
	if startupDebugEnabled {
		log.Printf("应用版本: %s", buildVersion)
		log.Printf(
			"Wails3 启动环境: GOOS=%s GOARCH=%s DISPLAY=%q WAYLAND_DISPLAY=%q XDG_SESSION_TYPE=%q XDG_CURRENT_DESKTOP=%q",
			goruntime.GOOS,
			goruntime.GOARCH,
			os.Getenv("DISPLAY"),
			os.Getenv("WAYLAND_DISPLAY"),
			os.Getenv("XDG_SESSION_TYPE"),
			os.Getenv("XDG_CURRENT_DESKTOP"),
		)
	}
	if startupDebugEnabled && goruntime.GOOS == "linux" && strings.TrimSpace(os.Getenv("DISPLAY")) == "" && strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY")) == "" {
		log.Printf("检测到 Linux 图形环境变量为空：DISPLAY / WAYLAND_DISPLAY 都未设置，GUI 窗口大概率无法创建")
	}

	cfg, err := backend.LoadConfig(backend.ResolveRuntimePath(appRoot, "config.yaml"))
	if err != nil {
		log.Printf("加载配置失败，使用默认配置: %v", err)
		cfg = backend.DefaultConfig()
	}

	app := NewApp(appRoot, buildVersion)
	windowWidth, windowHeight := normalizeStartupWindowSize(cfg.App.Window.Width, cfg.App.Window.Height)
	if state, err := backend.LoadWindowState(appRoot); err == nil {
		windowWidth, windowHeight = normalizeStartupWindowSize(state.Width, state.Height)
	}
	windowWidth, windowHeight = clampStartupWindowSize(windowWidth, windowHeight, cfg.App.Window.MinWidth, cfg.App.Window.MinHeight)

	var (
		wailsApp       *application.App
		mainWindow     application.Window
		startupReached = make(chan struct{})
		startupOnce    sync.Once
	)

	if startupDebugEnabled {
		go func() {
			select {
			case <-startupReached:
				return
			case <-time.After(12 * time.Second):
				log.Printf("Wails3 ApplicationStarted 在 12 秒内未触发。若终端一直转圈但没有窗口，优先检查图形环境和 WebView 依赖")
			}
		}()
	}

	shutdownContext := func() context.Context {
		if wailsApp != nil && wailsApp.Context() != nil {
			return wailsApp.Context()
		}
		return context.Background()
	}

	protoIPC, err := backend.NewWails3ProtoIPC(app.App, shutdownContext)
	if err != nil {
		log.Printf("Wails3 Proto IPC 二进制服务启动失败，将仅保留 raw message 兜底: %v", err)
	}
	rawMessageHandler := backend.NewWails3ProtoRawMessageHandler(shutdownContext)
	if protoIPC != nil {
		rawMessageHandler = protoIPC.RawMessageHandler()
		defer protoIPC.Close()
	}

	wailsApp = application.New(application.Options{
		Name:        cfg.App.Name,
		Description: "Trace Browser",
		Icon:        appIcon,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		OnShutdown: func() {
			if startupDebugEnabled {
				log.Printf("Wails3 OnShutdown 已触发")
			}
			backend.QuitTray()
			if protoIPC != nil {
				protoIPC.Close()
			}
			backend.Stop(app.App, shutdownContext())
		},
		RawMessageHandler: rawMessageHandler,
		ShouldQuit: func() bool {
			return !backend.ShouldBlockClose(app.App, shutdownContext())
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	wailsRuntime := backend.ConfigureWails3Runtime(app.App, wailsApp, nil)
	app.SetWindowSyncToolbarAdapter(&wails3WindowSyncToolbarAdapter{
		wailsApp:            wailsApp,
		protoIPC:            protoIPC,
		startupDebugEnabled: startupDebugEnabled,
	})

	mainWindow = wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            cfg.App.Name,
		Width:            windowWidth,
		Height:           windowHeight,
		MinWidth:         cfg.App.Window.MinWidth,
		MinHeight:        cfg.App.Window.MinHeight,
		BackgroundType:   application.BackgroundTypeSolid,
		BackgroundColour: application.NewRGBA(245, 247, 250, 255),
		EnableFileDrop:   true,
		URL:              "/",
	})
	wailsRuntime.SetWindow(mainWindow)

	mainWindow.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		if event == nil {
			return
		}
		ctx := event.Context()
		files := ctx.DroppedFiles()
		if len(files) == 0 {
			return
		}
		x, y := 0, 0
		if details := ctx.DropTargetDetails(); details != nil {
			x = details.X
			y = details.Y
		}
		app.EmitFileDropEvent(files, x, y)
	})

	mainWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if backend.ShouldBlockClose(app.App, shutdownContext()) {
			event.Cancel()
		}
	})

	wailsApp.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(event *application.ApplicationEvent) {
		startupOnce.Do(func() {
			close(startupReached)
			if startupDebugEnabled {
				log.Printf("Wails3 ApplicationStarted 已触发")
			}
			mainWindow.Center()
			if protoIPC != nil {
				protoIPC.InjectConfig(mainWindow)
			}
			go backend.RunTray(backend.TrayCallbacks{
				OnShow: func() {
					mainWindow.Show()
					mainWindow.UnMinimise()
				},
				OnQuitAppOnly: func() {
					app.QuitAppOnly()
				},
				OnQuit: func() {
					app.ForceQuit()
				},
			})
			backend.Start(app.App, shutdownContext())
			if startupDebugEnabled {
				log.Printf("后端 startup 已完成")
			}
		})
	})

	if startupDebugEnabled {
		log.Printf("准备调用 Wails3 application.Run 创建 GUI 窗口")
	}
	if err := wailsApp.Run(); err != nil {
		log.Fatal("启动 Wails3 应用失败:", err)
	}
	if startupDebugEnabled {
		log.Printf("Wails3 application.Run 已退出")
	}
}
