package platform

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails3Runtime adapts the Wails3 application/window APIs to the project platform boundary.
type Wails3Runtime struct {
	mu     sync.RWMutex
	app    *application.App
	window application.Window
}

func NewWails3Runtime(app *application.App, window application.Window) *Wails3Runtime {
	return &Wails3Runtime{
		app:    app,
		window: window,
	}
}

func (r *Wails3Runtime) SetWindow(window application.Window) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.window = window
	r.mu.Unlock()
}

func (r *Wails3Runtime) LogFatal(ctx context.Context, message string) {
	if r != nil && r.app != nil {
		r.app.Logger.Error(message)
		r.app.Quit()
		return
	}
	log.Fatal(message)
}

func (r *Wails3Runtime) EventsEmit(ctx context.Context, eventName string, optionalData ...any) {
	if r == nil || r.app == nil {
		return
	}
	r.app.Event.Emit(eventName, optionalData...)
}

func (r *Wails3Runtime) Quit(ctx context.Context) {
	if r == nil || r.app == nil {
		return
	}
	r.app.Quit()
}

func (r *Wails3Runtime) BrowserOpenURL(ctx context.Context, url string) {
	if r == nil || r.app == nil {
		return
	}
	_ = r.app.Browser.OpenURL(url)
}

func (r *Wails3Runtime) OpenFileDialog(ctx context.Context, options OpenDialogOptions) (string, error) {
	if r == nil || r.app == nil {
		return "", errors.New("wails3 application is not initialized")
	}
	return r.app.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		CanChooseFiles:                  true,
		CanChooseDirectories:            false,
		CanCreateDirectories:            options.CanCreateDirectories,
		ShowHiddenFiles:                 options.ShowHiddenFiles,
		ResolvesAliases:                 options.ResolvesAliases,
		TreatsFilePackagesAsDirectories: options.TreatPackagesAsDirectories,
		Filters:                         wails3FileFilters(options.Filters),
		Window:                          r.currentWindow(),
		Title:                           options.Title,
		Directory:                       options.DefaultDirectory,
	}).PromptForSingleSelection()
}

func (r *Wails3Runtime) OpenDirectoryDialog(ctx context.Context, options OpenDialogOptions) (string, error) {
	if r == nil || r.app == nil {
		return "", errors.New("wails3 application is not initialized")
	}
	return r.app.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		CanChooseFiles:                  false,
		CanChooseDirectories:            true,
		CanCreateDirectories:            options.CanCreateDirectories,
		ShowHiddenFiles:                 options.ShowHiddenFiles,
		ResolvesAliases:                 options.ResolvesAliases,
		TreatsFilePackagesAsDirectories: options.TreatPackagesAsDirectories,
		Filters:                         wails3FileFilters(options.Filters),
		Window:                          r.currentWindow(),
		Title:                           options.Title,
		Directory:                       options.DefaultDirectory,
	}).PromptForSingleSelection()
}

func (r *Wails3Runtime) SaveFileDialog(ctx context.Context, options SaveDialogOptions) (string, error) {
	if r == nil || r.app == nil {
		return "", errors.New("wails3 application is not initialized")
	}
	return r.app.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		CanCreateDirectories:            options.CanCreateDirectories,
		ShowHiddenFiles:                 options.ShowHiddenFiles,
		TreatsFilePackagesAsDirectories: options.TreatPackagesAsDirectories,
		Title:                           options.Title,
		Directory:                       options.DefaultDirectory,
		Filename:                        options.DefaultFilename,
		Filters:                         wails3FileFilters(options.Filters),
		Window:                          r.currentWindow(),
	}).PromptForSingleSelection()
}

func (r *Wails3Runtime) WindowIsNormal(ctx context.Context) bool {
	window := r.currentWindow()
	if window == nil {
		return false
	}
	return !window.IsMinimised() && !window.IsMaximised() && !window.IsFullscreen()
}

func (r *Wails3Runtime) WindowIsMaximised(ctx context.Context) bool {
	window := r.currentWindow()
	if window == nil {
		return false
	}
	return window.IsMaximised()
}

func (r *Wails3Runtime) WindowIsMinimised(ctx context.Context) bool {
	window := r.currentWindow()
	if window == nil {
		return false
	}
	return window.IsMinimised()
}

func (r *Wails3Runtime) WindowGetSize(ctx context.Context) (int, int) {
	window := r.currentWindow()
	if window == nil {
		return 0, 0
	}
	return window.Size()
}

func (r *Wails3Runtime) WindowGetPosition(ctx context.Context) (int, int) {
	window := r.currentWindow()
	if window == nil {
		return 0, 0
	}
	return window.Position()
}

func (r *Wails3Runtime) WindowHide(ctx context.Context) {
	window := r.currentWindow()
	if window != nil {
		window.Hide()
	}
}

func (r *Wails3Runtime) WindowMinimise(ctx context.Context) {
	window := r.currentWindow()
	if window != nil {
		window.Minimise()
	}
}

func (r *Wails3Runtime) currentWindow() application.Window {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.window
}

func wails3FileFilters(filters []FileFilter) []application.FileFilter {
	if len(filters) == 0 {
		return nil
	}
	out := make([]application.FileFilter, 0, len(filters))
	for _, filter := range filters {
		out = append(out, application.FileFilter{
			DisplayName: filter.DisplayName,
			Pattern:     filter.Pattern,
		})
	}
	return out
}
