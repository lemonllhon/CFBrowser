//go:build !windows
// +build !windows

package backend

type windowSyncToolbarController struct{}

func (c *windowSyncToolbarController) Show(app *App, state *WindowSyncState) error {
	return nil
}

func (c *windowSyncToolbarController) Update(state *WindowSyncState) error {
	return nil
}

func (c *windowSyncToolbarController) Hide() error {
	return nil
}
