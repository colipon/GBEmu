package joypad

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/colipon/gbemu/internal/mmu"
	"github.com/colipon/gbemu/internal/sgb"
)

// Button indices
const (
	BtnRight = iota
	BtnLeft
	BtnUp
	BtnDown
	BtnA
	BtnB
	BtnSelect
	BtnStart
)

// Joypad tracks key state and exposes the FF00 register
type Joypad struct {
	mem     *mmu.MMU
	SGB     *sgb.SGB // nil = DMG mode
	buttons [8]bool
}

// New creates and registers a Joypad. Pass a non-nil SGB to enable packet detection.
func New(mem *mmu.MMU, s *sgb.SGB) *Joypad {
	j := &Joypad{mem: mem, SGB: s}
	mem.OnJoypadRead = j.Read
	if s != nil {
		mem.OnJoypadWrite = s.UpdateJoypad
	}
	return j
}

// Update polls Ebiten key state each frame
func (j *Joypad) Update() {
	prev := j.buttons

	j.buttons[BtnRight]  = ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	j.buttons[BtnLeft]   = ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	j.buttons[BtnUp]     = ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	j.buttons[BtnDown]   = ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	j.buttons[BtnA]      = ebiten.IsKeyPressed(ebiten.KeyZ)
	j.buttons[BtnB]      = ebiten.IsKeyPressed(ebiten.KeyX)
	j.buttons[BtnSelect] = ebiten.IsKeyPressed(ebiten.KeyBackspace)
	j.buttons[BtnStart]  = ebiten.IsKeyPressed(ebiten.KeyEnter)

	// Request interrupt on any new press
	for i := 0; i < 8; i++ {
		if j.buttons[i] && !prev[i] {
			j.mem.RequestInterrupt(mmu.IntJoypad)
			break
		}
	}
}

// Read returns the current value of the joypad register (FF00)
func (j *Joypad) Read() byte {
	sel := j.mem.IO[0x00]
	result := byte(0xFF)

	if sel&0x20 == 0 { // Action buttons
		if j.buttons[BtnA]      { result &^= 0x01 }
		if j.buttons[BtnB]      { result &^= 0x02 }
		if j.buttons[BtnSelect] { result &^= 0x04 }
		if j.buttons[BtnStart]  { result &^= 0x08 }
	}
	if sel&0x10 == 0 { // Direction buttons
		if j.buttons[BtnRight] { result &^= 0x01 }
		if j.buttons[BtnLeft]  { result &^= 0x02 }
		if j.buttons[BtnUp]    { result &^= 0x04 }
		if j.buttons[BtnDown]  { result &^= 0x08 }
	}

	return result
}
