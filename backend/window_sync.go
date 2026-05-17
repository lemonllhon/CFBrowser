package backend

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const defaultWindowSyncMasterColor = "#2563eb"

type WindowSyncLayoutSettings struct {
	Mode     string `json:"mode"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	GapX     int    `json:"gapX"`
	GapY     int    `json:"gapY"`
	PerRow   int    `json:"perRow"`
	UpdatedAt string `json:"updatedAt"`
}

type WindowSyncCandidate struct {
	ProfileId    string `json:"profileId"`
	ProfileName  string `json:"profileName"`
	DebugPort    int    `json:"debugPort"`
	Pid          int    `json:"pid"`
	Running      bool   `json:"running"`
	DebugReady   bool   `json:"debugReady"`
	Role         string `json:"role"`
	Master       bool   `json:"master"`
	CanSync      bool   `json:"canSync"`
	Unavailable string `json:"unavailable"`
}

type WindowSyncStartInput struct {
	ProfileIds      []string `json:"profileIds"`
	MasterProfileId string   `json:"masterProfileId"`
}

type WindowSyncState struct {
	SessionId       string                `json:"sessionId"`
	Active          bool                  `json:"active"`
	Paused          bool                  `json:"paused"`
	MasterProfileId string                `json:"masterProfileId"`
	ProfileIds      []string              `json:"profileIds"`
	Windows         []WindowSyncCandidate `json:"windows"`
	MasterColor     string                `json:"masterColor"`
	Layout          WindowSyncLayoutSettings `json:"layout"`
	StartedAt       string                `json:"startedAt"`
	UpdatedAt       string                `json:"updatedAt"`
}

func (a *App) WindowSyncListCandidates() []WindowSyncCandidate {
	if a == nil || a.browserMgr == nil {
		return []WindowSyncCandidate{}
	}

	activeState := a.WindowSyncGetState()
	activeById := make(map[string]WindowSyncCandidate)
	if activeState != nil && activeState.Active {
		for _, item := range activeState.Windows {
			activeById[item.ProfileId] = item
		}
	}

	a.browserMgr.Mutex.Lock()
	items := make([]WindowSyncCandidate, 0, len(a.browserMgr.Profiles))
	for _, profile := range a.browserMgr.Profiles {
		if profile == nil {
			continue
		}
		item := WindowSyncCandidate{
			ProfileId:   profile.ProfileId,
			ProfileName: profile.ProfileName,
			DebugPort:   profile.DebugPort,
			Pid:         profile.Pid,
			Running:     profile.Running,
			DebugReady:  profile.DebugReady,
			CanSync:     profile.Running && profile.DebugReady && profile.DebugPort > 0,
		}
		if !profile.Running {
			item.Unavailable = "实例未运行"
		} else if !profile.DebugReady || profile.DebugPort <= 0 {
			item.Unavailable = "调试端口未就绪"
		} else if !canConnectDebugPort(profile.DebugPort, 250*time.Millisecond) {
			item.CanSync = false
			item.Unavailable = "调试端口不可连接"
		}
		if active, ok := activeById[item.ProfileId]; ok {
			item.Role = active.Role
			item.Master = active.Master
		}
		if item.CanSync {
			items = append(items, item)
		}
	}
	a.browserMgr.Mutex.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.Compare(items[i].ProfileName, items[j].ProfileName) < 0
	})
	return items
}

func (a *App) WindowSyncStart(input WindowSyncStartInput) (*WindowSyncState, error) {
	if a == nil || a.browserMgr == nil {
		return nil, fmt.Errorf("浏览器管理器未就绪")
	}

	profileIds := normalizeWindowSyncProfileIds(input.ProfileIds)
	if len(profileIds) < 2 {
		return nil, fmt.Errorf("至少选择 2 个运行中的窗口")
	}
	masterProfileId := strings.TrimSpace(input.MasterProfileId)
	if masterProfileId == "" {
		return nil, fmt.Errorf("请选择主控窗口")
	}
	if !containsString(profileIds, masterProfileId) {
		return nil, fmt.Errorf("主控窗口必须在已选窗口中")
	}

	windows, err := a.resolveWindowSyncCandidates(profileIds, masterProfileId)
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	state := &WindowSyncState{
		SessionId:       uuid.NewString(),
		Active:          true,
		Paused:          false,
		MasterProfileId: masterProfileId,
		ProfileIds:      profileIds,
		Windows:         windows,
		MasterColor:     defaultWindowSyncMasterColor,
		Layout:          a.normalizedWindowSyncLayoutSettings(a.windowSyncLayout),
		StartedAt:       now,
		UpdatedAt:       now,
	}

	a.windowSyncMu.Lock()
	a.windowSyncState = cloneWindowSyncState(state)
	a.windowSyncMu.Unlock()

	if err := a.applyWindowSyncLayoutToState(state.Layout, state); err != nil {
		a.windowSyncMu.Lock()
		a.windowSyncState = nil
		a.windowSyncMu.Unlock()
		return nil, err
	}

	a.emitWindowSyncStateChanged(state)
	return cloneWindowSyncState(state), nil
}

func (a *App) WindowSyncApplyLayout(input WindowSyncLayoutSettings) (*WindowSyncState, error) {
	settings := a.normalizedWindowSyncLayoutSettings(input)

	a.windowSyncMu.Lock()
	if a.windowSyncState == nil || !a.windowSyncState.Active {
		a.windowSyncMu.Unlock()
		return nil, fmt.Errorf("窗口同步未启动")
	}
	now := time.Now().Format(time.RFC3339)
	settings.UpdatedAt = now
	a.windowSyncLayout = settings
	a.windowSyncState.Layout = settings
	a.windowSyncState.UpdatedAt = now
	state := cloneWindowSyncState(a.windowSyncState)
	a.windowSyncMu.Unlock()

	if err := a.applyWindowSyncLayoutToState(settings, state); err != nil {
		return state, err
	}
	a.emitWindowSyncStateChanged(state)
	return state, nil
}

func (a *App) WindowSyncSaveLayoutSettings(input WindowSyncLayoutSettings) (*WindowSyncLayoutSettings, error) {
	settings := a.normalizedWindowSyncLayoutSettings(input)
	settings.UpdatedAt = time.Now().Format(time.RFC3339)

	a.windowSyncMu.Lock()
	a.windowSyncLayout = settings
	if a.windowSyncState != nil && a.windowSyncState.Active {
		a.windowSyncState.Layout = settings
		a.windowSyncState.UpdatedAt = settings.UpdatedAt
	}
	a.windowSyncMu.Unlock()

	return &settings, nil
}

func (a *App) WindowSyncGetLayoutSettings() WindowSyncLayoutSettings {
	if a == nil {
		return defaultWindowSyncLayoutSettings()
	}
	a.windowSyncMu.Lock()
	defer a.windowSyncMu.Unlock()
	if a.windowSyncState != nil && a.windowSyncState.Active && strings.TrimSpace(a.windowSyncState.Layout.Mode) != "" {
		return a.normalizedWindowSyncLayoutSettings(a.windowSyncState.Layout)
	}
	return a.normalizedWindowSyncLayoutSettings(a.windowSyncLayout)
}

func (a *App) WindowSyncGetState() *WindowSyncState {
	if a == nil {
		return nil
	}
	a.windowSyncMu.Lock()
	defer a.windowSyncMu.Unlock()
	return cloneWindowSyncState(a.windowSyncState)
}

func (a *App) WindowSyncStop() (*WindowSyncState, error) {
	if a == nil {
		return nil, nil
	}
	a.windowSyncMu.Lock()
	previous := cloneWindowSyncState(a.windowSyncState)
	a.windowSyncState = nil
	a.windowSyncMu.Unlock()

	if previous != nil {
		previous.Active = false
		previous.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	a.emitWindowSyncStateChanged(previous)
	return previous, nil
}

func (a *App) WindowSyncPause() (*WindowSyncState, error) {
	state, err := a.updateWindowSyncPaused(true)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (a *App) WindowSyncResume() (*WindowSyncState, error) {
	state, err := a.updateWindowSyncPaused(false)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (a *App) WindowSyncShowAll() (*WindowSyncState, error) {
	state, err := a.requireWindowSyncState()
	if err != nil {
		return nil, err
	}

	failures := make([]string, 0)
	for _, item := range state.Windows {
		if err := a.showWindowSyncProfile(item.ProfileId); err != nil {
			name := strings.TrimSpace(item.ProfileName)
			if name == "" {
				name = item.ProfileId
			}
			failures = append(failures, fmt.Sprintf("%s：%v", name, err))
		}
	}
	if len(failures) > 0 {
		return state, fmt.Errorf("部分窗口弹出失败：%s", strings.Join(failures, "；"))
	}
	return state, nil
}

func (a *App) ensureWindowSyncProfileMutable(profileId string) error {
	if a == nil {
		return nil
	}
	profileId = strings.TrimSpace(profileId)
	if profileId == "" {
		return nil
	}
	a.windowSyncMu.Lock()
	defer a.windowSyncMu.Unlock()
	if a.windowSyncState == nil || !a.windowSyncState.Active {
		return nil
	}
	if a.windowSyncState.MasterProfileId == profileId {
		return fmt.Errorf("同步状态下无法修改主控窗口，请先停止窗口同步")
	}
	return nil
}

func (a *App) updateWindowSyncPaused(paused bool) (*WindowSyncState, error) {
	if a == nil {
		return nil, fmt.Errorf("窗口同步未启动")
	}
	a.windowSyncMu.Lock()
	if a.windowSyncState == nil || !a.windowSyncState.Active {
		a.windowSyncMu.Unlock()
		return nil, fmt.Errorf("窗口同步未启动")
	}
	a.windowSyncState.Paused = paused
	a.windowSyncState.UpdatedAt = time.Now().Format(time.RFC3339)
	state := cloneWindowSyncState(a.windowSyncState)
	a.windowSyncMu.Unlock()

	a.emitWindowSyncStateChanged(state)
	return state, nil
}

func (a *App) requireWindowSyncState() (*WindowSyncState, error) {
	if a == nil {
		return nil, fmt.Errorf("窗口同步未启动")
	}
	a.windowSyncMu.Lock()
	defer a.windowSyncMu.Unlock()
	if a.windowSyncState == nil || !a.windowSyncState.Active {
		return nil, fmt.Errorf("窗口同步未启动")
	}
	return cloneWindowSyncState(a.windowSyncState), nil
}

func (a *App) resolveWindowSyncCandidates(profileIds []string, masterProfileId string) ([]WindowSyncCandidate, error) {
	a.browserMgr.Mutex.Lock()
	defer a.browserMgr.Mutex.Unlock()

	windows := make([]WindowSyncCandidate, 0, len(profileIds))
	for _, profileId := range profileIds {
		profile, exists := a.browserMgr.Profiles[profileId]
		if !exists || profile == nil {
			return nil, fmt.Errorf("实例不存在：%s", profileId)
		}
		if !profile.Running {
			return nil, fmt.Errorf("实例未运行：%s", profile.ProfileName)
		}
		if !profile.DebugReady || profile.DebugPort <= 0 {
			return nil, fmt.Errorf("实例调试端口未就绪：%s", profile.ProfileName)
		}
		if !canConnectDebugPort(profile.DebugPort, 250*time.Millisecond) {
			return nil, fmt.Errorf("实例调试端口不可连接：%s", profile.ProfileName)
		}
		isMaster := profile.ProfileId == masterProfileId
		role := "controlled"
		if isMaster {
			role = "master"
		}
		windows = append(windows, WindowSyncCandidate{
			ProfileId:   profile.ProfileId,
			ProfileName: profile.ProfileName,
			DebugPort:   profile.DebugPort,
			Pid:         profile.Pid,
			Running:     profile.Running,
			DebugReady:  profile.DebugReady,
			Role:        role,
			Master:      isMaster,
			CanSync:     true,
		})
	}
	return windows, nil
}

func (a *App) pinWindowSyncMasterTopLeft(masterProfileId string) error {
	a.browserMgr.Mutex.Lock()
	profile, exists := a.browserMgr.Profiles[masterProfileId]
	if !exists || profile == nil {
		a.browserMgr.Mutex.Unlock()
		return fmt.Errorf("主控实例不存在")
	}
	debugPort := profile.DebugPort
	pid := profile.Pid
	a.browserMgr.Mutex.Unlock()

	targetID, err := firstPageTargetID(debugPort)
	if err != nil {
		return fmt.Errorf("主控窗口定位失败: %w", err)
	}
	windowInfo, err := cdpBrowserCallResult(debugPort, "Browser.getWindowForTarget", map[string]any{"targetId": targetID})
	if err != nil {
		return fmt.Errorf("主控窗口信息获取失败: %w", err)
	}
	windowID, ok := numericResult(windowInfo["windowId"])
	if !ok {
		return fmt.Errorf("无法获取主控窗口 ID")
	}

	area := primaryWorkArea()
	width, height := browserWindowSizeFromBounds(windowInfo["bounds"], area)
	left := area.Left
	top := area.Top

	if _, err := cdpBrowserCallResult(debugPort, "Browser.setWindowBounds", map[string]any{
		"windowId": windowID,
		"bounds": map[string]any{
			"windowState": "normal",
			"left":        left,
			"top":         top,
			"width":       width,
			"height":      height,
		},
	}); err != nil {
		return fmt.Errorf("主控窗口移动失败: %w", err)
	}
	_, _ = cdpCall(debugPort, "Page.bringToFront", nil)
	if pid > 0 {
		_ = setBrowserWindowsTopmostByPID(pid, left, top, width, height)
	}
	return nil
}

func (a *App) showWindowSyncProfile(profileId string) error {
	profileId = strings.TrimSpace(profileId)
	if profileId == "" {
		return fmt.Errorf("profile id is required")
	}

	a.browserMgr.Mutex.Lock()
	profile, exists := a.browserMgr.Profiles[profileId]
	if !exists || profile == nil {
		a.browserMgr.Mutex.Unlock()
		return fmt.Errorf("实例不存在")
	}
	if !profile.Running || !profile.DebugReady || profile.DebugPort <= 0 {
		a.browserMgr.Mutex.Unlock()
		return fmt.Errorf("实例未运行或调试端口未就绪")
	}
	debugPort := profile.DebugPort
	pid := profile.Pid
	a.browserMgr.Mutex.Unlock()

	targetID, err := firstPageTargetID(debugPort)
	if err != nil {
		return err
	}
	windowInfo, err := cdpBrowserCallResult(debugPort, "Browser.getWindowForTarget", map[string]any{"targetId": targetID})
	if err != nil {
		return err
	}
	windowID, ok := numericResult(windowInfo["windowId"])
	if !ok {
		return fmt.Errorf("无法获取浏览器窗口 ID")
	}

	bounds := map[string]any{"windowState": "normal"}
	if rawBounds, ok := windowInfo["bounds"].(map[string]any); ok {
		for _, key := range []string{"left", "top", "width", "height"} {
			if value, ok := numericResult(rawBounds[key]); ok {
				bounds[key] = value
			}
		}
	}
	if _, err := cdpBrowserCallResult(debugPort, "Browser.setWindowBounds", map[string]any{
		"windowId": windowID,
		"bounds":   bounds,
	}); err != nil {
		return err
	}
	_, _ = cdpCall(debugPort, "Page.bringToFront", nil)

	if pid > 0 {
		left, top, width, height := 0, 0, 0, 0
		if value, ok := numericResult(bounds["left"]); ok {
			left = value
		}
		if value, ok := numericResult(bounds["top"]); ok {
			top = value
		}
		if value, ok := numericResult(bounds["width"]); ok {
			width = value
		}
		if value, ok := numericResult(bounds["height"]); ok {
			height = value
		}
		if width > 0 && height > 0 {
			_ = setBrowserWindowsTopmostByPID(pid, left, top, width, height)
		}
	}
	return nil
}

func (a *App) applyWindowSyncLayoutToState(settings WindowSyncLayoutSettings, state *WindowSyncState) error {
	if state == nil || len(state.Windows) == 0 {
		return fmt.Errorf("窗口同步未启动")
	}
	rects := calculateWindowSyncLayoutRects(settings, orderedWindowSyncWindows(state), primaryWorkArea())
	failures := make([]string, 0)
	for _, item := range orderedWindowSyncWindows(state) {
		rect, ok := rects[item.ProfileId]
		if !ok {
			continue
		}
		if err := a.setWindowSyncProfileBounds(item.ProfileId, rect); err != nil {
			name := strings.TrimSpace(item.ProfileName)
			if name == "" {
				name = item.ProfileId
			}
			failures = append(failures, fmt.Sprintf("%s：%v", name, err))
		}
	}

	if strings.EqualFold(strings.TrimSpace(settings.Mode), "stack") {
		_ = a.showWindowSyncProfile(state.MasterProfileId)
	}
	if len(failures) > 0 {
		return fmt.Errorf("部分窗口布局失败：%s", strings.Join(failures, "；"))
	}
	return nil
}

func (a *App) setWindowSyncProfileBounds(profileId string, rect workAreaRect) error {
	profileId = strings.TrimSpace(profileId)
	if profileId == "" {
		return fmt.Errorf("profile id is required")
	}

	a.browserMgr.Mutex.Lock()
	profile, exists := a.browserMgr.Profiles[profileId]
	if !exists || profile == nil {
		a.browserMgr.Mutex.Unlock()
		return fmt.Errorf("实例不存在")
	}
	if !profile.Running || !profile.DebugReady || profile.DebugPort <= 0 {
		a.browserMgr.Mutex.Unlock()
		return fmt.Errorf("实例未运行或调试端口未就绪")
	}
	debugPort := profile.DebugPort
	pid := profile.Pid
	a.browserMgr.Mutex.Unlock()

	targetID, err := firstPageTargetID(debugPort)
	if err != nil {
		return err
	}
	windowInfo, err := cdpBrowserCallResult(debugPort, "Browser.getWindowForTarget", map[string]any{"targetId": targetID})
	if err != nil {
		return err
	}
	windowID, ok := numericResult(windowInfo["windowId"])
	if !ok {
		return fmt.Errorf("无法获取浏览器窗口 ID")
	}
	if _, err := cdpBrowserCallResult(debugPort, "Browser.setWindowBounds", map[string]any{
		"windowId": windowID,
		"bounds": map[string]any{
			"windowState": "normal",
			"left":        rect.Left,
			"top":         rect.Top,
			"width":       rect.Width,
			"height":      rect.Height,
		},
	}); err != nil {
		return err
	}
	_, _ = cdpCall(debugPort, "Page.bringToFront", nil)
	if pid > 0 {
		_ = setBrowserWindowsTopmostByPID(pid, rect.Left, rect.Top, rect.Width, rect.Height)
	}
	return nil
}

func calculateWindowSyncLayoutRects(settings WindowSyncLayoutSettings, windows []WindowSyncCandidate, area workAreaRect) map[string]workAreaRect {
	settings = normalizeWindowSyncLayoutSettings(settings)
	rects := make(map[string]workAreaRect, len(windows))
	count := len(windows)
	if count == 0 {
		return rects
	}

	switch strings.ToLower(strings.TrimSpace(settings.Mode)) {
	case "stack":
		for _, item := range windows {
			rects[item.ProfileId] = workAreaRect{
				Left:   area.Left,
				Top:    area.Top,
				Width:  maxInt(320, area.Width),
				Height: maxInt(240, area.Height),
			}
		}
	case "custom":
		perRow := maxInt(1, settings.PerRow)
		width := maxInt(320, settings.Width)
		height := maxInt(240, settings.Height)
		for index, item := range windows {
			col := index % perRow
			row := index / perRow
			rects[item.ProfileId] = workAreaRect{
				Left:   area.Left + col*(width+settings.GapX),
				Top:    area.Top + row*(height+settings.GapY),
				Width:  width,
				Height: height,
			}
		}
	default:
		cols := int(math.Ceil(math.Sqrt(float64(count))))
		if cols < 1 {
			cols = 1
		}
		rows := int(math.Ceil(float64(count) / float64(cols)))
		if rows < 1 {
			rows = 1
		}
		gapX, gapY := 8, 8
		width := (area.Width - gapX*(cols-1)) / cols
		height := (area.Height - gapY*(rows-1)) / rows
		width = maxInt(320, width)
		height = maxInt(240, height)
		for index, item := range windows {
			col := index % cols
			row := index / cols
			rects[item.ProfileId] = workAreaRect{
				Left:   area.Left + col*(width+gapX),
				Top:    area.Top + row*(height+gapY),
				Width:  width,
				Height: height,
			}
		}
	}
	return rects
}

func orderedWindowSyncWindows(state *WindowSyncState) []WindowSyncCandidate {
	if state == nil {
		return nil
	}
	windows := append([]WindowSyncCandidate{}, state.Windows...)
	sort.SliceStable(windows, func(i, j int) bool {
		if windows[i].ProfileId == state.MasterProfileId {
			return true
		}
		if windows[j].ProfileId == state.MasterProfileId {
			return false
		}
		return i < j
	})
	return windows
}

func (a *App) normalizedWindowSyncLayoutSettings(settings WindowSyncLayoutSettings) WindowSyncLayoutSettings {
	return normalizeWindowSyncLayoutSettings(settings)
}

func normalizeWindowSyncLayoutSettings(settings WindowSyncLayoutSettings) WindowSyncLayoutSettings {
	mode := strings.ToLower(strings.TrimSpace(settings.Mode))
	if mode != "grid" && mode != "stack" && mode != "custom" {
		mode = "grid"
	}
	out := WindowSyncLayoutSettings{
		Mode:      mode,
		Width:     settings.Width,
		Height:    settings.Height,
		GapX:      maxInt(0, settings.GapX),
		GapY:      maxInt(0, settings.GapY),
		PerRow:    settings.PerRow,
		UpdatedAt: settings.UpdatedAt,
	}
	if out.Width <= 0 {
		out.Width = 1500
	}
	if out.Height <= 0 {
		out.Height = 500
	}
	if out.PerRow <= 0 {
		out.PerRow = 2
	}
	return out
}

func defaultWindowSyncLayoutSettings() WindowSyncLayoutSettings {
	return WindowSyncLayoutSettings{
		Mode:   "grid",
		Width:  1500,
		Height: 500,
		GapX:   10,
		GapY:   10,
		PerRow: 2,
	}
}

func (a *App) emitWindowSyncStateChanged(state *WindowSyncState) {
	if a == nil || a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "window-sync:state-changed", state)
}

func cloneWindowSyncState(state *WindowSyncState) *WindowSyncState {
	if state == nil {
		return nil
	}
	snapshot := *state
	snapshot.ProfileIds = append([]string{}, state.ProfileIds...)
	snapshot.Windows = append([]WindowSyncCandidate{}, state.Windows...)
	return &snapshot
}

func normalizeWindowSyncProfileIds(profileIds []string) []string {
	seen := make(map[string]struct{}, len(profileIds))
	out := make([]string, 0, len(profileIds))
	for _, profileId := range profileIds {
		value := strings.TrimSpace(profileId)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
