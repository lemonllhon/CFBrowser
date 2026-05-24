package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const windowStatePath = "data/window-state.json"

type WindowState struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

func LoadWindowState(appRoot string) (WindowState, error) {
	path := ResolveRuntimePath(appRoot, windowStatePath)
	data, err := os.ReadFile(path)
	if err != nil {
		return WindowState{}, err
	}

	var state WindowState
	if err := json.Unmarshal(data, &state); err != nil {
		return WindowState{}, err
	}
	return state, nil
}

func (a *App) SaveWindowState(width int, height int) error {
	state := WindowState{
		Width:     width,
		Height:    height,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
	if state.Width <= 0 || state.Height <= 0 {
		return fmt.Errorf("窗口尺寸无效: %dx%d", state.Width, state.Height)
	}

	path := a.resolveAppPath(windowStatePath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (a *App) saveCurrentWindowState(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if !a.appRuntime().WindowIsNormal(ctx) {
		return nil
	}
	width, height := a.appRuntime().WindowGetSize(ctx)
	return a.SaveWindowState(width, height)
}
