package app

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// WailsWindowAdapter adapts a Wails WebviewWindow to the services.Window interface.
type WailsWindowAdapter struct {
	win *application.WebviewWindow
}

// NewWailsWindowAdapter creates a new WailsWindowAdapter for the given window.
func NewWailsWindowAdapter(win *application.WebviewWindow) *WailsWindowAdapter {
	return &WailsWindowAdapter{win: win}
}

func (w *WailsWindowAdapter) IsFullscreen() bool {
	if w == nil || w.win == nil {
		return false
	}
	return w.win.IsFullscreen()
}

func (w *WailsWindowAdapter) IsMaximised() bool {
	if w == nil || w.win == nil {
		return false
	}
	return w.win.IsMaximised()
}

func (w *WailsWindowAdapter) Size() (int, int) {
	if w == nil || w.win == nil {
		return 0, 0
	}
	return w.win.Size()
}

func (w *WailsWindowAdapter) Fullscreen() {
	if w == nil || w.win == nil {
		return
	}
	w.win.Fullscreen()
}

func (w *WailsWindowAdapter) UnFullscreen() {
	if w == nil || w.win == nil {
		return
	}
	w.win.UnFullscreen()
}

func (w *WailsWindowAdapter) OnResize(fn func()) {
	if w == nil || w.win == nil {
		return
	}
	w.win.OnWindowEvent(events.Common.WindowDidResize, func(event *application.WindowEvent) {
		fn()
	})
}

func (w *WailsWindowAdapter) OnClose(fn func()) {
	if w == nil || w.win == nil {
		return
	}
	w.win.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		fn()
	})
}
