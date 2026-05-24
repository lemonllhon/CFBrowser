package backend

import (
	"ant-chrome/backend/internal/platform"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Wails3RuntimeHandle struct {
	runtime *platform.Wails3Runtime
}

func ConfigureWails3Runtime(app *App, wailsApp *application.App, window application.Window) *Wails3RuntimeHandle {
	runtime := platform.NewWails3Runtime(wailsApp, window)
	if app != nil {
		app.useRuntime(runtime)
	}
	return &Wails3RuntimeHandle{runtime: runtime}
}

func (h *Wails3RuntimeHandle) SetWindow(window application.Window) {
	if h == nil || h.runtime == nil {
		return
	}
	h.runtime.SetWindow(window)
}
