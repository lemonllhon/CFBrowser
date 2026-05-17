package backend

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const defaultWindowSyncMasterColor = "#2563eb"

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
		StartedAt:       now,
		UpdatedAt:       now,
	}

	a.windowSyncMu.Lock()
	a.windowSyncState = cloneWindowSyncState(state)
	a.windowSyncMu.Unlock()

	if err := a.pinWindowSyncMasterTopLeft(masterProfileId); err != nil {
		a.windowSyncMu.Lock()
		a.windowSyncState = nil
		a.windowSyncMu.Unlock()
		return nil, err
	}

	a.emitWindowSyncStateChanged(state)
	return cloneWindowSyncState(state), nil
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
