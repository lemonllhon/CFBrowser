package backend

import (
	"ant-chrome/backend/internal/logger"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const defaultWindowSyncMasterColor = "#2563eb"

const windowSyncBindingName = "__traceWindowSyncEvent"

type windowSyncTarget struct {
	Id           string
	Title        string
	Url          string
	Attached     bool
	Index        int
	WebSocketURL string
}

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
	CanAutoStart bool   `json:"canAutoStart"`
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
	SyncKeyboard    bool                  `json:"syncKeyboard"`
	SyncMouse       bool                  `json:"syncMouse"`
	Layout          WindowSyncLayoutSettings `json:"layout"`
	StartedAt       string                `json:"startedAt"`
	UpdatedAt       string                `json:"updatedAt"`
}

type WindowSyncSettings struct {
	MasterColor  string `json:"masterColor"`
	SyncKeyboard bool   `json:"syncKeyboard"`
	SyncMouse    bool   `json:"syncMouse"`
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
			CanAutoStart: !profile.Running,
		}
		if !profile.Running {
			item.Unavailable = "未运行，将在开始同步时自动启动"
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
		if item.CanSync || item.CanAutoStart {
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

	if err := a.ensureWindowSyncProfilesReady(profileIds); err != nil {
		return nil, err
	}

	windows, err := a.resolveWindowSyncCandidates(profileIds, masterProfileId)
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	layout := a.normalizedWindowSyncLayoutSettings(a.windowSyncLayout)
	layout.Mode = "grid"
	layout.UpdatedAt = now
	state := &WindowSyncState{
		SessionId:       uuid.NewString(),
		Active:          true,
		Paused:          false,
		MasterProfileId: masterProfileId,
		ProfileIds:      profileIds,
		Windows:         windows,
		MasterColor:     defaultWindowSyncMasterColor,
		SyncKeyboard:    true,
		SyncMouse:       true,
		Layout:          layout,
		StartedAt:       now,
		UpdatedAt:       now,
	}

	a.windowSyncMu.Lock()
	a.stopWindowSyncListenerLocked()
	a.windowSyncState = cloneWindowSyncState(state)
	a.windowSyncMu.Unlock()

	if err := a.applyWindowSyncLayoutToState(state.Layout, state); err != nil {
		a.windowSyncMu.Lock()
		a.windowSyncState = nil
		a.windowSyncMu.Unlock()
		return nil, err
	}

	a.emitWindowSyncStateChanged(state)
	a.showWindowSyncToolbar(state)
	a.startWindowSyncListener()
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
	a.updateWindowSyncToolbar(state)
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
	a.stopWindowSyncListenerLocked()
	a.windowSyncState = nil
	a.windowSyncMu.Unlock()

	if previous != nil {
		previous.Active = false
		previous.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	a.hideWindowSyncToolbar()
	a.emitWindowSyncStateChanged(previous)
	return previous, nil
}

func (a *App) WindowSyncGetSettings() WindowSyncSettings {
	state := a.WindowSyncGetState()
	if state != nil && state.Active {
		return WindowSyncSettings{
			MasterColor:  state.MasterColor,
			SyncKeyboard: state.SyncKeyboard,
			SyncMouse:    state.SyncMouse,
		}
	}
	return defaultWindowSyncSettings()
}

func (a *App) WindowSyncSaveSettings(input WindowSyncSettings) (*WindowSyncState, error) {
	settings := normalizeWindowSyncSettings(input)
	a.windowSyncMu.Lock()
	if a.windowSyncState == nil || !a.windowSyncState.Active {
		a.windowSyncMu.Unlock()
		return nil, fmt.Errorf("窗口同步未启动")
	}
	a.windowSyncState.MasterColor = settings.MasterColor
	a.windowSyncState.SyncKeyboard = settings.SyncKeyboard
	a.windowSyncState.SyncMouse = settings.SyncMouse
	a.windowSyncState.UpdatedAt = time.Now().Format(time.RFC3339)
	state := cloneWindowSyncState(a.windowSyncState)
	a.windowSyncMu.Unlock()

	a.emitWindowSyncStateChanged(state)
	a.updateWindowSyncToolbar(state)
	return state, nil
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
	a.updateWindowSyncToolbar(state)
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
	a.updateWindowSyncToolbar(state)
	return state, nil
}

func (a *App) startWindowSyncListener() {
	if a == nil {
		return
	}
	a.windowSyncMu.Lock()
	if a.windowSyncState == nil || !a.windowSyncState.Active {
		a.windowSyncMu.Unlock()
		return
	}
	a.windowSyncSeq++
	seq := a.windowSyncSeq
	cancel := make(chan struct{})
	a.windowSyncCancel = cancel
	a.windowSyncMu.Unlock()

	go a.runWindowSyncListener(seq, cancel)
}

func (a *App) stopWindowSyncListenerLocked() {
	if a.windowSyncCancel != nil {
		close(a.windowSyncCancel)
		a.windowSyncCancel = nil
	}
}

func (a *App) runWindowSyncListener(seq int, cancel <-chan struct{}) {
	lastActiveTab := ""
	for {
		select {
		case <-cancel:
			return
		default:
		}

		state := a.WindowSyncGetState()
		if state == nil || !state.Active {
			return
		}
		master := findWindowSyncWindow(state.Windows, state.MasterProfileId)
		if master == nil || master.DebugPort <= 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if err := a.listenWindowSyncMaster(seq, cancel, state, master.DebugPort, &lastActiveTab); err != nil {
			select {
			case <-cancel:
				return
			case <-time.After(700 * time.Millisecond):
			}
		}
	}
}

func (a *App) listenWindowSyncMaster(seq int, cancel <-chan struct{}, state *WindowSyncState, debugPort int, lastActiveTab *string) error {
	targets, err := pageWebSocketTargets(debugPort)
	if err != nil {
		return err
	}
	for index := range targets {
		targets[index].Index = index
	}
	localCancel := make(chan struct{})
	defer close(localCancel)
	done := make(chan error, 1)
	connected := 0
	for _, target := range targets {
		target := target
		go func() {
			if err := a.listenWindowSyncTarget(seq, cancel, localCancel, target); err != nil {
				select {
				case done <- err:
				default:
				}
			}
		}()
		connected++
	}
	logger.New("WindowSync").Info("主控窗口同步监听已启动",
		logger.F("debug_port", debugPort),
		logger.F("targets", connected),
		logger.F("session_id", state.SessionId),
	)
	if connected == 0 {
		return fmt.Errorf("未找到可监听的主控页面")
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	refreshAfter := 0
	for {
		select {
		case <-cancel:
			return nil
		case err := <-done:
			logger.New("WindowSync").Warn("主控页面监听中断", logger.F("error", err.Error()))
			return err
		case <-ticker.C:
			refreshAfter++
			a.syncWindowSyncTabs(seq, debugPort, lastActiveTab)
			if refreshAfter >= 4 {
				return nil
			}
		}
	}
}

func (a *App) listenWindowSyncTarget(seq int, cancel <-chan struct{}, localCancel <-chan struct{}, target windowSyncTarget) error {
	conn, _, err := websocket.DefaultDialer.Dial(target.WebSocketURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	nextID := 1
	send := func(method string, params map[string]any) error {
		msg := cdpMessage{Id: nextID, Method: method, Params: params}
		nextID++
		return conn.WriteJSON(msg)
	}
	if err := send("Runtime.enable", nil); err != nil {
		return err
	}
	if err := send("Page.enable", nil); err != nil {
		return err
	}
	_ = send("Runtime.removeBinding", map[string]any{"name": windowSyncBindingName})
	if err := send("Runtime.addBinding", map[string]any{"name": windowSyncBindingName}); err != nil && !strings.Contains(strings.ToLower(err.Error()), "already") {
		return err
	}
	source := windowSyncInjectionScript(target)
	if err := send("Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": source}); err != nil {
		return err
	}
	if err := send("Runtime.evaluate", map[string]any{"expression": source, "awaitPromise": false}); err != nil {
		return err
	}
	activationSource := windowSyncActivationScript(target)
	if err := send("Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": activationSource}); err != nil {
		return err
	}
	if err := send("Runtime.evaluate", map[string]any{"expression": activationSource, "awaitPromise": false}); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		for {
			var raw map[string]any
			if err := conn.ReadJSON(&raw); err != nil {
				done <- err
				return
			}
			if method, _ := raw["method"].(string); method == "Runtime.bindingCalled" {
				params, _ := raw["params"].(map[string]any)
				payload, _ := params["payload"].(string)
				a.handleWindowSyncPayload(seq, payload)
			}
		}
	}()

	select {
	case <-cancel:
		_ = conn.Close()
		return nil
	case <-localCancel:
		_ = conn.Close()
		return nil
	case err := <-done:
		return fmt.Errorf("%s: %w", target.Id, err)
	}
}

func (a *App) handleWindowSyncPayload(seq int, payload string) {
	if strings.TrimSpace(payload) == "" {
		return
	}
	var event windowSyncEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return
	}
	state := a.WindowSyncGetState()
	if state == nil || !state.Active || state.Paused {
		return
	}
	a.windowSyncMu.Lock()
	currentSeq := a.windowSyncSeq
	a.windowSyncMu.Unlock()
	if currentSeq != seq {
		return
	}

	isKeyboard := event.Type == "keyDown" || event.Type == "keyUp" || event.Type == "input"
	isMouse := event.Type == "wheel" || event.Type == "mouseDown" || event.Type == "mouseMove" || event.Type == "mouseUp" || event.Type == "tabActivated"
	if isKeyboard && !state.SyncKeyboard {
		return
	}
	if isMouse && !state.SyncMouse {
		return
	}

	for _, item := range state.Windows {
		if item.ProfileId == state.MasterProfileId {
			continue
		}
		if item.DebugPort <= 0 {
			continue
		}
		if err := dispatchWindowSyncEvent(item.DebugPort, event); err != nil {
			logger.New("WindowSync").Warn("同步事件派发失败",
				logger.F("profile_id", item.ProfileId),
				logger.F("event_type", event.Type),
				logger.F("error", err.Error()),
			)
		}
	}
}

func (a *App) syncWindowSyncTabs(seq int, masterDebugPort int, lastActiveTab *string) {
	state := a.WindowSyncGetState()
	if state == nil || !state.Active || state.Paused || !state.SyncMouse {
		return
	}
	a.windowSyncMu.Lock()
	currentSeq := a.windowSyncSeq
	a.windowSyncMu.Unlock()
	if currentSeq != seq {
		return
	}

	masterTargets, err := pageWebSocketTargets(masterDebugPort)
	if err != nil || len(masterTargets) == 0 {
		return
	}
	_, activeTarget := activeWindowSyncTarget(masterTargets)
	if strings.TrimSpace(activeTarget.Id) == "" {
		return
	}
	activeKey := normalizeWindowSyncTargetURL(activeTarget.Url)
	if activeKey == "" {
		activeKey = strings.TrimSpace(activeTarget.Id)
	}
	if lastActiveTab != nil && *lastActiveTab == activeKey {
		return
	}
	if lastActiveTab != nil {
		*lastActiveTab = activeKey
	}

	for _, item := range state.Windows {
		if item.ProfileId == state.MasterProfileId || item.DebugPort <= 0 {
			continue
		}
		if err := syncWindowSyncTabsToControlled(item.DebugPort, activeTarget); err != nil {
			logger.New("WindowSync").Warn("同步标签页失败",
				logger.F("profile_id", item.ProfileId),
				logger.F("error", err.Error()),
			)
		}
	}
}

type windowSyncEvent struct {
	Type        string  `json:"type"`
	TargetId    string  `json:"targetId"`
	TargetIndex int     `json:"targetIndex"`
	TargetUrl   string  `json:"targetUrl"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Button      string  `json:"button"`
	Buttons     int     `json:"buttons"`
	DeltaX      float64 `json:"deltaX"`
	DeltaY      float64 `json:"deltaY"`
	Key         string  `json:"key"`
	Code        string  `json:"code"`
	Text        string  `json:"text"`
	Value       string  `json:"value"`
	Modifiers   int     `json:"modifiers"`
}

func dispatchWindowSyncEvent(debugPort int, event windowSyncEvent) error {
	targets, err := pageWebSocketTargets(debugPort)
	if err != nil {
		return err
	}
	if event.Type == "tabActivated" {
		target, err := ensureWindowSyncTargetForEvent(debugPort, targets, event)
		if err != nil {
			return err
		}
		return activateWindowSyncTarget(debugPort, target)
	}
	target := findWindowSyncTargetForEvent(targets, event)
	if strings.TrimSpace(target.WebSocketURL) != "" {
		return dispatchWindowSyncEventToTarget(target.WebSocketURL, event)
	}
	var lastErr error
	dispatched := 0
	for _, target := range targets {
		if err := dispatchWindowSyncEventToTarget(target.WebSocketURL, event); err != nil {
			lastErr = err
			continue
		}
		dispatched++
	}
	if dispatched == 0 && lastErr != nil {
		return lastErr
	}
	if dispatched == 0 {
		return fmt.Errorf("未找到可派发的页面")
	}
	return nil
}

func dispatchWindowSyncEventToTarget(wsURL string, event windowSyncEvent) error {
	switch event.Type {
	case "mouseDown", "mouseUp", "mouseMove":
		button := normalizeWindowSyncMouseButton(event.Button)
		cdpType := "mouseMoved"
		switch event.Type {
		case "mouseDown":
			cdpType = "mousePressed"
		case "mouseUp":
			cdpType = "mouseReleased"
		}
		params := map[string]any{
			"type":       cdpType,
			"x":          event.X,
			"y":          event.Y,
			"button":     button,
			"modifiers":  event.Modifiers,
		}
		if event.Buttons > 0 {
			params["buttons"] = event.Buttons
		}
		if event.Type == "mouseDown" || event.Type == "mouseUp" {
			params["clickCount"] = 1
		}
		if _, err := cdpCallWebSocket(wsURL, "Input.dispatchMouseEvent", params); err != nil {
			return err
		}
		return nil
	case "wheel":
		_, err := cdpCallWebSocket(wsURL, "Input.dispatchMouseEvent", map[string]any{
			"type":       "mouseWheel",
			"x":          event.X,
			"y":          event.Y,
			"deltaX":     event.DeltaX,
			"deltaY":     event.DeltaY,
			"modifiers":  event.Modifiers,
		})
		return err
	case "keyDown", "keyUp":
		cdpType := "keyDown"
		if event.Type == "keyUp" {
			cdpType = "keyUp"
		}
		params := map[string]any{
			"type":                  cdpType,
			"key":                   event.Key,
			"code":                  event.Code,
			"windowsVirtualKeyCode": windowSyncVirtualKeyCode(event),
			"nativeVirtualKeyCode":  windowSyncVirtualKeyCode(event),
			"modifiers":             event.Modifiers,
		}
		if event.Type == "keyDown" && strings.TrimSpace(event.Text) != "" {
			params["text"] = event.Text
			params["unmodifiedText"] = event.Text
		}
		_, err := cdpCallWebSocket(wsURL, "Input.dispatchKeyEvent", params)
		return err
	case "input":
		expression := fmt.Sprintf(`(() => {
  const el = document.activeElement;
  if (!el || !("value" in el)) return false;
  const value = %q;
  el.focus();
  el.value = value;
  el.dispatchEvent(new Event("input", { bubbles: true }));
  el.dispatchEvent(new Event("change", { bubbles: true }));
  try {
    if (typeof el.setSelectionRange === "function") el.setSelectionRange(value.length, value.length);
  } catch (_) {}
  return true;
})()`, event.Value)
		_, err := cdpCallWebSocket(wsURL, "Runtime.evaluate", map[string]any{"expression": expression, "awaitPromise": false})
		return err
	default:
		return nil
	}
}

func cdpCallWebSocket(wsURL string, method string, params map[string]any) (map[string]any, error) {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("WebSocket 连接失败: %w", err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	msg := cdpMessage{Id: 1, Method: method, Params: params}
	if err := conn.WriteJSON(msg); err != nil {
		return nil, fmt.Errorf("CDP 命令发送失败: %w", err)
	}

	var cdpResp cdpResponse
	if err := conn.ReadJSON(&cdpResp); err != nil {
		return nil, fmt.Errorf("CDP 响应读取失败: %w", err)
	}
	if cdpResp.Error != nil {
		return nil, fmt.Errorf("CDP 错误: %s", cdpResp.Error.Message)
	}
	return cdpResp.Result, nil
}

func activateWindowSyncTarget(debugPort int, target windowSyncTarget) error {
	targetID := strings.TrimSpace(target.Id)
	if targetID == "" {
		return fmt.Errorf("缺少标签页 target id")
	}
	if _, err := cdpBrowserCallResult(debugPort, "Target.activateTarget", map[string]any{"targetId": targetID}); err != nil {
		return err
	}
	if target.WebSocketURL != "" {
		_, _ = cdpCallWebSocket(target.WebSocketURL, "Page.bringToFront", nil)
	}
	return nil
}

func syncWindowSyncTabsToControlled(debugPort int, activeTarget windowSyncTarget) error {
	targets, err := pageWebSocketTargets(debugPort)
	if err != nil {
		return err
	}
	target := findWindowSyncTargetByURL(targets, activeTarget.Url)
	if strings.TrimSpace(target.Id) == "" {
		target, err = createWindowSyncTarget(debugPort, activeTarget.Url)
		if err != nil {
			return err
		}
	}
	return activateWindowSyncTarget(debugPort, target)
}

func ensureWindowSyncTargetForEvent(debugPort int, targets []windowSyncTarget, event windowSyncEvent) (windowSyncTarget, error) {
	if target := findWindowSyncTargetByURL(targets, event.TargetUrl); strings.TrimSpace(target.Id) != "" {
		return target, nil
	}
	if strings.TrimSpace(event.TargetUrl) != "" {
		return createWindowSyncTarget(debugPort, event.TargetUrl)
	}
	if event.TargetIndex >= 0 {
		for len(targets) <= event.TargetIndex {
			target, err := createWindowSyncTarget(debugPort, "about:blank")
			if err != nil {
				return windowSyncTarget{}, err
			}
			targets = append(targets, target)
		}
	}
	target := findWindowSyncTargetForEvent(targets, event)
	if strings.TrimSpace(target.Id) == "" {
		return windowSyncTarget{}, fmt.Errorf("被控窗口缺少同序标签页：%d", event.TargetIndex+1)
	}
	return target, nil
}

func activeWindowSyncTarget(targets []windowSyncTarget) (int, windowSyncTarget) {
	for index, target := range targets {
		if target.WebSocketURL == "" {
			continue
		}
		active, err := cdpEvaluateBool(target.WebSocketURL, `document.visibilityState === "visible"`)
		if err == nil && active {
			return index, target
		}
	}
	for index, target := range targets {
		if strings.TrimSpace(target.Id) != "" {
			return index, target
		}
	}
	return -1, windowSyncTarget{}
}

func cdpEvaluateBool(wsURL string, expression string) (bool, error) {
	result, err := cdpCallWebSocket(wsURL, "Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
	})
	if err != nil {
		return false, err
	}
	remote, _ := result["result"].(map[string]any)
	value, _ := remote["value"].(bool)
	return value, nil
}

func findWindowSyncTargetForEvent(targets []windowSyncTarget, event windowSyncEvent) windowSyncTarget {
	if target := findWindowSyncTargetByURL(targets, event.TargetUrl); strings.TrimSpace(target.Id) != "" {
		return target
	}
	if event.TargetIndex >= 0 && event.TargetIndex < len(targets) {
		return targets[event.TargetIndex]
	}
	return windowSyncTarget{}
}

func findWindowSyncTargetByURL(targets []windowSyncTarget, targetURL string) windowSyncTarget {
	normalizedURL := normalizeWindowSyncTargetURL(targetURL)
	if normalizedURL == "" {
		return windowSyncTarget{}
	}
	for _, target := range targets {
		if normalizeWindowSyncTargetURL(target.Url) == normalizedURL {
			return target
		}
	}
	return windowSyncTarget{}
}

func createWindowSyncTarget(debugPort int, rawURL string) (windowSyncTarget, error) {
	targetURL := strings.TrimSpace(rawURL)
	if targetURL == "" {
		targetURL = "about:blank"
	}
	created, err := cdpBrowserCallResult(debugPort, "Target.createTarget", map[string]any{"url": targetURL})
	if err != nil {
		return windowSyncTarget{}, err
	}
	createdID, _ := created["targetId"].(string)
	targets, err := pageWebSocketTargets(debugPort)
	if err != nil {
		return windowSyncTarget{}, err
	}
	for _, target := range targets {
		if target.Id == createdID {
			return target, nil
		}
	}
	if target := findWindowSyncTargetByURL(targets, targetURL); strings.TrimSpace(target.Id) != "" {
		return target, nil
	}
	return windowSyncTarget{}, fmt.Errorf("新建标签页后未找到目标：%s", targetURL)
}

func normalizeWindowSyncTargetURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" {
		return strings.TrimRight(strings.ToLower(rawURL), "/")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	if (parsed.Scheme == "https" && strings.HasSuffix(parsed.Host, ":443")) || (parsed.Scheme == "http" && strings.HasSuffix(parsed.Host, ":80")) {
		host, _, splitErr := strings.Cut(parsed.Host, ":")
		if splitErr {
			parsed.Host = host
		}
	}
	if parsed.Path == "/" {
		parsed.Path = ""
	}
	return strings.TrimRight(parsed.String(), "/")
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

func (a *App) ensureWindowSyncProfilesReady(profileIds []string) error {
	for _, profileId := range profileIds {
		profile, err := a.windowSyncProfileSnapshot(profileId)
		if err != nil {
			return err
		}
		if profile.Running && profile.DebugReady && profile.DebugPort > 0 && canConnectDebugPort(profile.DebugPort, 250*time.Millisecond) {
			continue
		}
		if profile.Running {
			return fmt.Errorf("实例调试端口未就绪：%s", profile.ProfileName)
		}
		if _, err := a.browserInstanceStartInternal(profileId, nil, nil, false, true); err != nil {
			return fmt.Errorf("自动启动实例失败：%s，%w", profile.ProfileName, err)
		}
		if err := a.waitWindowSyncProfileReady(profileId, 30*time.Second); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) windowSyncProfileSnapshot(profileId string) (BrowserProfile, error) {
	a.browserMgr.Mutex.Lock()
	defer a.browserMgr.Mutex.Unlock()
	profile, exists := a.browserMgr.Profiles[profileId]
	if !exists || profile == nil {
		return BrowserProfile{}, fmt.Errorf("实例不存在：%s", profileId)
	}
	return *profile, nil
}

func (a *App) waitWindowSyncProfileReady(profileId string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		profile, err := a.windowSyncProfileSnapshot(profileId)
		if err != nil {
			return err
		}
		if profile.Running && profile.DebugReady && profile.DebugPort > 0 && canConnectDebugPort(profile.DebugPort, 250*time.Millisecond) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("自动启动后调试端口未就绪：%s", profile.ProfileName)
		}
		time.Sleep(300 * time.Millisecond)
	}
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
	return nil
}

func (a *App) applyWindowSyncLayoutToState(settings WindowSyncLayoutSettings, state *WindowSyncState) error {
	if state == nil || len(state.Windows) == 0 {
		return fmt.Errorf("窗口同步未启动")
	}
	rects := calculateWindowSyncLayoutRects(settings, orderedWindowSyncWindows(state), a.windowSyncLayoutWorkArea())
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

func (a *App) windowSyncLayoutWorkArea() workAreaRect {
	if x, y, ok := appWindowCenterPoint(); ok {
		return workAreaForPoint(x, y)
	}
	return primaryWorkArea()
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
		offset := 24
		width := maxInt(320, area.Width)
		height := maxInt(240, area.Height)
		for index, item := range windows {
			left := area.Left + index*offset
			top := area.Top + index*offset
			if left+width > area.Left+area.Width {
				left = area.Left
			}
			if top+height > area.Top+area.Height {
				top = area.Top
			}
			rects[item.ProfileId] = workAreaRect{
				Left:   left,
				Top:    top,
				Width:  width,
				Height: height,
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
		cols, rows := bestWindowSyncGrid(count, area)
		gapX, gapY := 8, 8
		width := maxInt(160, (area.Width-gapX*(cols-1))/cols)
		height := maxInt(120, (area.Height-gapY*(rows-1))/rows)
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

func bestWindowSyncGrid(count int, area workAreaRect) (int, int) {
	if count <= 1 {
		return 1, 1
	}
	screenRatio := float64(maxInt(1, area.Width)) / float64(maxInt(1, area.Height))
	bestCols, bestRows := count, 1
	bestScore := math.MaxFloat64
	for cols := 1; cols <= count; cols++ {
		rows := int(math.Ceil(float64(count) / float64(cols)))
		cellRatio := (float64(area.Width) / float64(cols)) / (float64(area.Height) / float64(rows))
		score := math.Abs(math.Log(cellRatio))
		if cols < rows && screenRatio >= 1 {
			score += 0.2
		}
		if score < bestScore {
			bestScore = score
			bestCols = cols
			bestRows = rows
		}
	}
	return bestCols, bestRows
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

func pageWebSocketTargets(debugPort int) ([]windowSyncTarget, error) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/json", debugPort))
	if err != nil {
		return nil, fmt.Errorf("CDP /json 请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var targets []cdpTarget
	if err := json.Unmarshal(body, &targets); err != nil || len(targets) == 0 {
		return nil, fmt.Errorf("CDP targets 解析失败或为空")
	}
	out := make([]windowSyncTarget, 0, len(targets))
	for _, target := range targets {
		wsURL := strings.TrimSpace(target.WebSocketDebuggerUrl)
		if target.Type != "page" || wsURL == "" {
			continue
		}
		out = append(out, windowSyncTarget{
			Id:           strings.TrimSpace(target.Id),
			Title:        strings.TrimSpace(target.Title),
			Url:          strings.TrimSpace(target.Url),
			Attached:     target.Attached,
			WebSocketURL: wsURL,
		})
	}
	if len(out) > 0 {
		return out, nil
	}
	for _, target := range targets {
		wsURL := strings.TrimSpace(target.WebSocketDebuggerUrl)
		if wsURL == "" {
			continue
		}
		out = append(out, windowSyncTarget{
			Id:           strings.TrimSpace(target.Id),
			Title:        strings.TrimSpace(target.Title),
			Url:          strings.TrimSpace(target.Url),
			Attached:     target.Attached,
			WebSocketURL: wsURL,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("未找到可用的 WebSocket 调试地址")
	}
	return out, nil
}

func windowSyncInjectionScript(target windowSyncTarget) string {
	return fmt.Sprintf(`(() => {
  if (window.__traceWindowSyncInstalled) return;
  window.__traceWindowSyncInstalled = true;
  const targetId = %q;
  const targetIndex = %d;
  const targetUrl = %q;
  const send = (event) => {
    try {
      const fn = window[%q];
      if (typeof fn === "function") fn(JSON.stringify(event));
    } catch (_) {}
  };
  const modifiers = (event) => (event.altKey ? 1 : 0) | (event.ctrlKey ? 2 : 0) | (event.metaKey ? 4 : 0) | (event.shiftKey ? 8 : 0);
  const textFor = (event) => {
    if (!event.key || event.ctrlKey || event.metaKey || event.altKey) return "";
    return event.key.length === 1 ? event.key : "";
  };
  const mouseBase = (event) => ({
    targetId,
    targetIndex,
    targetUrl: window.location.href,
    x: event.clientX,
    y: event.clientY,
    button: event.button === 2 ? "right" : event.button === 1 ? "middle" : "left",
    buttons: event.buttons || 0,
    modifiers: modifiers(event)
  });
  window.addEventListener("mousedown", (event) => send({
    type: "mouseDown",
    ...mouseBase(event)
  }), true);
  window.addEventListener("mousemove", (event) => {
    if (!event.buttons) return;
    send({
      type: "mouseMove",
      ...mouseBase(event)
    });
  }, true);
  window.addEventListener("mouseup", (event) => send({
    type: "mouseUp",
    ...mouseBase(event)
  }), true);
  window.addEventListener("wheel", (event) => send({
    type: "wheel",
    targetId,
    targetIndex,
    targetUrl: window.location.href,
    x: event.clientX,
    y: event.clientY,
    deltaX: event.deltaX,
    deltaY: event.deltaY,
    modifiers: modifiers(event)
  }), { capture: true, passive: true });
  window.addEventListener("input", (event) => {
    const target = event.target;
    if (!target || !("value" in target)) return;
    send({
      type: "input",
      targetId,
      targetIndex,
      targetUrl: window.location.href,
      value: String(target.value ?? "")
    });
  }, true);
  window.addEventListener("keydown", (event) => send({
    type: "keyDown",
    targetId,
    targetIndex,
    targetUrl: window.location.href,
    key: event.key,
    code: event.code,
    text: textFor(event),
    modifiers: modifiers(event)
  }), true);
  window.addEventListener("keyup", (event) => send({
    type: "keyUp",
    targetId,
    targetIndex,
    targetUrl: window.location.href,
    key: event.key,
    code: event.code,
    modifiers: modifiers(event)
  }), true);
})();`, target.Id, target.Index, target.Url, windowSyncBindingName)
}

func windowSyncActivationScript(target windowSyncTarget) string {
	payload, _ := json.Marshal(windowSyncEvent{
		Type:        "tabActivated",
		TargetId:    target.Id,
		TargetIndex: target.Index,
	})
	return fmt.Sprintf(`(() => {
  if (window.__traceWindowSyncActivationInstalled) return;
  window.__traceWindowSyncActivationInstalled = true;
  const basePayload = %s;
  let timer = null;
  const send = () => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      if (document.visibilityState !== "visible") return;
      try {
        const fn = window[%q];
        if (typeof fn === "function") {
          fn(JSON.stringify({ ...basePayload, targetUrl: window.location.href }));
        }
      } catch (_) {}
    }, 150);
  };
  if (document.visibilityState === "visible") send();
  window.addEventListener("focus", send, true);
  document.addEventListener("visibilitychange", send, true);
})();`, string(payload), windowSyncBindingName)
}

func normalizeWindowSyncMouseButton(button string) string {
	switch strings.ToLower(strings.TrimSpace(button)) {
	case "right":
		return "right"
	case "middle":
		return "middle"
	default:
		return "left"
	}
}

func windowSyncVirtualKeyCode(event windowSyncEvent) int {
	key := strings.TrimSpace(event.Key)
	if len([]rune(key)) == 1 {
		r := []rune(strings.ToUpper(key))[0]
		if r >= 'A' && r <= 'Z' {
			return int(r)
		}
		if r >= '0' && r <= '9' {
			return int(r)
		}
	}
	switch key {
	case "Backspace":
		return 8
	case "Tab":
		return 9
	case "Enter":
		return 13
	case "Shift":
		return 16
	case "Control":
		return 17
	case "Alt":
		return 18
	case "Escape":
		return 27
	case " ":
		return 32
	case "ArrowLeft":
		return 37
	case "ArrowUp":
		return 38
	case "ArrowRight":
		return 39
	case "ArrowDown":
		return 40
	case "Delete":
		return 46
	default:
		return 0
	}
}

func findWindowSyncWindow(windows []WindowSyncCandidate, profileId string) *WindowSyncCandidate {
	for i := range windows {
		if windows[i].ProfileId == profileId {
			item := windows[i]
			return &item
		}
	}
	return nil
}

func defaultWindowSyncSettings() WindowSyncSettings {
	return WindowSyncSettings{
		MasterColor:  defaultWindowSyncMasterColor,
		SyncKeyboard: true,
		SyncMouse:    true,
	}
}

func normalizeWindowSyncSettings(input WindowSyncSettings) WindowSyncSettings {
	out := defaultWindowSyncSettings()
	out.MasterColor = normalizeWindowSyncMasterColor(input.MasterColor)
	out.SyncKeyboard = input.SyncKeyboard
	out.SyncMouse = input.SyncMouse
	return out
}

func normalizeWindowSyncMasterColor(color string) string {
	value := strings.TrimSpace(color)
	if value == "" {
		return defaultWindowSyncMasterColor
	}
	if !strings.HasPrefix(value, "#") {
		value = "#" + value
	}
	if len(value) == 4 {
		value = fmt.Sprintf("#%c%c%c%c%c%c", value[1], value[1], value[2], value[2], value[3], value[3])
	}
	if len(value) != 7 {
		return defaultWindowSyncMasterColor
	}
	for _, ch := range value[1:] {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			continue
		}
		return defaultWindowSyncMasterColor
	}
	return strings.ToLower(value)
}

func (a *App) emitWindowSyncStateChanged(state *WindowSyncState) {
	if a == nil || a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "window-sync:state-changed", state)
}

func (a *App) showWindowSyncToolbar(state *WindowSyncState) {
	if a == nil || state == nil || !state.Active {
		return
	}
	_ = a.windowSyncToolbar.Show(a, state)
}

func (a *App) updateWindowSyncToolbar(state *WindowSyncState) {
	if a == nil || state == nil || !state.Active {
		return
	}
	_ = a.windowSyncToolbar.Update(state)
}

func (a *App) hideWindowSyncToolbar() {
	if a == nil {
		return
	}
	_ = a.windowSyncToolbar.Hide()
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
