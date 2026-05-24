package platform

import "context"

// FileFilter defines a selectable file type in open/save dialogs.
type FileFilter struct {
	DisplayName string
	Pattern     string
}

// OpenDialogOptions contains the common options used by file and directory open dialogs.
type OpenDialogOptions struct {
	DefaultDirectory           string
	DefaultFilename            string
	Title                      string
	Filters                    []FileFilter
	ShowHiddenFiles            bool
	CanCreateDirectories       bool
	ResolvesAliases            bool
	TreatPackagesAsDirectories bool
}

// SaveDialogOptions contains the options used by save dialogs.
type SaveDialogOptions struct {
	DefaultDirectory           string
	DefaultFilename            string
	Title                      string
	Filters                    []FileFilter
	ShowHiddenFiles            bool
	CanCreateDirectories       bool
	TreatPackagesAsDirectories bool
}

// Runtime is the narrow platform boundary used by business services.
type Runtime interface {
	LogFatal(ctx context.Context, message string)
	EventsEmit(ctx context.Context, eventName string, optionalData ...any)
	Quit(ctx context.Context)
	BrowserOpenURL(ctx context.Context, url string)
	OpenFileDialog(ctx context.Context, options OpenDialogOptions) (string, error)
	OpenDirectoryDialog(ctx context.Context, options OpenDialogOptions) (string, error)
	SaveFileDialog(ctx context.Context, options SaveDialogOptions) (string, error)
	WindowIsNormal(ctx context.Context) bool
	WindowIsMaximised(ctx context.Context) bool
	WindowIsMinimised(ctx context.Context) bool
	WindowGetSize(ctx context.Context) (int, int)
	WindowGetPosition(ctx context.Context) (int, int)
	WindowHide(ctx context.Context)
	WindowMinimise(ctx context.Context)
}

var defaultRuntime Runtime

func DefaultRuntime() Runtime {
	if defaultRuntime == nil {
		return noopRuntime{}
	}
	return defaultRuntime
}

func SetDefaultRuntime(runtime Runtime) {
	if runtime != nil {
		defaultRuntime = runtime
	}
}

type noopRuntime struct{}

func (noopRuntime) LogFatal(ctx context.Context, message string) {}

func (noopRuntime) EventsEmit(ctx context.Context, eventName string, optionalData ...any) {}

func (noopRuntime) Quit(ctx context.Context) {}

func (noopRuntime) BrowserOpenURL(ctx context.Context, url string) {}

func (noopRuntime) OpenFileDialog(ctx context.Context, options OpenDialogOptions) (string, error) {
	return "", nil
}

func (noopRuntime) OpenDirectoryDialog(ctx context.Context, options OpenDialogOptions) (string, error) {
	return "", nil
}

func (noopRuntime) SaveFileDialog(ctx context.Context, options SaveDialogOptions) (string, error) {
	return "", nil
}

func (noopRuntime) WindowIsNormal(ctx context.Context) bool {
	return false
}

func (noopRuntime) WindowIsMaximised(ctx context.Context) bool {
	return false
}

func (noopRuntime) WindowIsMinimised(ctx context.Context) bool {
	return false
}

func (noopRuntime) WindowGetSize(ctx context.Context) (int, int) {
	return 0, 0
}

func (noopRuntime) WindowGetPosition(ctx context.Context) (int, int) {
	return 0, 0
}

func (noopRuntime) WindowHide(ctx context.Context) {}

func (noopRuntime) WindowMinimise(ctx context.Context) {}
