package debugger

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Window is an Ebiten Game that hosts the debugger UI
type Window struct {
	dbg *Debugger
}

// NewWindow wraps a Debugger in an Ebiten-compatible Game
func NewWindow(dbg *Debugger) *Window {
	return &Window{dbg: dbg}
}

func (w *Window) Update() error {
	w.dbg.Update()
	return nil
}

func (w *Window) Draw(screen *ebiten.Image) {
	w.dbg.Draw(screen)
}

func (w *Window) Layout(_, _ int) (int, int) {
	return WinW, WinH
}
