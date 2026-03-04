// Package debugger implements a BGB-style in-game debugger UI using Ebitengine.
// It renders in a separate Ebiten window and provides:
//   - Memory viewer + hex editor
//   - VRAM viewer (tiles, BG map, OAM)
//   - CPU register / flag view
//   - Breakpoints & step execution
package debugger

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/colipon/gbemu/internal/cpu"
	"github.com/colipon/gbemu/internal/mmu"
)

// Tab indices
const (
	TabMemory = iota
	TabVRAM
	TabOAM
	TabCPU
	tabCount
)

var tabNames = [tabCount]string{"Memory", "VRAM", "OAM", "CPU"}

// Window dimensions
const (
	WinW = 800
	WinH = 600

	charW = 6  // ebitenutil debug font char width
	charH = 14 // line height
	padding = 6
)

// Palette
var (
	colBG       = color.RGBA{0x1e, 0x1e, 0x2e, 0xFF}
	colPanel    = color.RGBA{0x28, 0x28, 0x3e, 0xFF}
	colTabActive = color.RGBA{0x45, 0x85, 0x88, 0xFF}
	colTabHover = color.RGBA{0x38, 0x38, 0x55, 0xFF}
	colText     = color.RGBA{0xcd, 0xd6, 0xf4, 0xFF}
	colDim      = color.RGBA{0x6c, 0x70, 0x86, 0xFF}
	colHex      = color.RGBA{0x89, 0xdc, 0xeb, 0xFF}
	colAddr     = color.RGBA{0xa6, 0xe3, 0xa1, 0xFF}
	colHighlight = color.RGBA{0xf3, 0x8b, 0xa8, 0xFF}
	colEditing  = color.RGBA{0xff, 0xe0, 0x80, 0xFF}
	colBreak    = color.RGBA{0xf3, 0x8b, 0xa8, 0xFF}
	colChanged  = color.RGBA{0xfa, 0xb3, 0x87, 0xFF}
)

// Debugger is the top-level debugger state
type Debugger struct {
	mem *mmu.MMU
	cpu *cpu.CPU

	activeTab int

	// Memory tab
	memBase    uint16 // top-left address
	memCursor  uint16 // selected byte address
	editMode   bool
	editNibble int    // 0 = high nibble, 1 = low nibble
	editBuf    string
	prevMem    [0x10000]byte // for change highlighting
	memChanged [0x10000]bool

	// VRAM tab
	vramZoom    int // 1 or 2
	vramPage    int // 0 = tiles $8000, 1 = tiles $8800, 2 = BG map $9800, 3 = BG map $9C00
	tileCanvas  *ebiten.Image

	// OAM tab
	oamSelected int

	// CPU tab
	breakpoints map[uint16]bool
	paused      bool
	stepOnce    bool

	// Scroll
	scrollY float64
}

// New creates a Debugger connected to the given MMU and CPU
func New(m *mmu.MMU, c *cpu.CPU) *Debugger {
	d := &Debugger{
		mem:         m,
		cpu:         c,
		vramZoom:    2,
		breakpoints: make(map[uint16]bool),
		tileCanvas:  ebiten.NewImage(128, 192), // 16×24 tiles × 8px
	}
	// snapshot initial memory
	for i := 0; i < 0x10000; i++ {
		d.prevMem[i] = m.Read(uint16(i))
	}
	return d
}

// IsPaused returns true when the emulator should not advance
func (d *Debugger) IsPaused() bool { return d.paused }

// ConsumeStep returns true once after a step request, then false
func (d *Debugger) ConsumeStep() bool {
	if d.stepOnce {
		d.stepOnce = false
		return true
	}
	return false
}

// CheckBreakpoint pauses execution if PC hits a breakpoint
func (d *Debugger) CheckBreakpoint(pc uint16) {
	if d.breakpoints[pc] {
		d.paused = true
	}
}

// Update handles input and updates debugger state — call once per Ebiten frame
func (d *Debugger) Update() {
	d.handleTabSwitch()

	switch d.activeTab {
	case TabMemory:
		d.updateMemory()
	case TabVRAM:
		d.updateVRAM()
	case TabOAM:
		d.updateOAM()
	case TabCPU:
		d.updateCPU()
	}

	// Snapshot memory changes
	for i := 0; i < 0x10000; i++ {
		cur := d.mem.Read(uint16(i))
		d.memChanged[i] = cur != d.prevMem[i]
		d.prevMem[i] = cur
	}
}

func (d *Debugger) handleTabSwitch() {
	for i := 0; i < tabCount; i++ {
		if inpututil.IsKeyJustPressed(ebiten.KeyF1 + ebiten.Key(i)) {
			d.activeTab = i
		}
	}
	// Mouse click on tab bar
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if my >= padding && my <= padding+charH+4 {
			tabW := (WinW - padding*2) / tabCount
			tab := (mx - padding) / tabW
			if tab >= 0 && tab < tabCount {
				d.activeTab = tab
			}
		}
	}
}

// Draw renders the debugger onto screen
func (d *Debugger) Draw(screen *ebiten.Image) {
	screen.Fill(colBG)
	d.drawTabBar(screen)

	contentY := padding*2 + charH + 4
	switch d.activeTab {
	case TabMemory:
		d.drawMemory(screen, contentY)
	case TabVRAM:
		d.drawVRAM(screen, contentY)
	case TabOAM:
		d.drawOAM(screen, contentY)
	case TabCPU:
		d.drawCPU(screen, contentY)
	}
}

func (d *Debugger) drawTabBar(screen *ebiten.Image) {
	tabW := (WinW - padding*2) / tabCount
	for i, name := range tabNames {
		x := padding + i*tabW
		y := padding
		bg := colPanel
		if i == d.activeTab {
			bg = colTabActive
		}
		drawRect(screen, x, y, tabW-2, charH+4, bg)
		label := fmt.Sprintf("%s [F%d]", name, i+1)
		ebitenutil.DebugPrintAt(screen, label, x+4, y+2)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func drawRect(dst *ebiten.Image, x, y, w, h int, c color.RGBA) {
	img := ebiten.NewImage(w, h)
	img.Fill(c)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(img, op)
}

func printAt(dst *ebiten.Image, s string, x, y int, c color.RGBA) {
	// Ebiten's built-in debug font is white; we use ColorScale to tint
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	tmp := ebiten.NewImage(len(s)*charW+1, charH)
	ebitenutil.DebugPrint(tmp, s)
	dst.DrawImage(tmp, op)
}

func hexByte(b byte) string { return fmt.Sprintf("%02X", b) }
func hexWord(w uint16) string { return fmt.Sprintf("%04X", w) }

// ── Memory tab ────────────────────────────────────────────────────────────────

const memCols = 16
const memRows = 28

func (d *Debugger) updateMemory() {
	// Navigate with arrow keys
	if !d.editMode {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) { d.memCursor++ }
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft)  { d.memCursor-- }
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown)  { d.memCursor += memCols }
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp)    { d.memCursor -= memCols }
		if inpututil.IsKeyJustPressed(ebiten.KeyPageDown)   { d.memCursor += uint16(memCols * memRows) }
		if inpututil.IsKeyJustPressed(ebiten.KeyPageUp)     { d.memCursor -= uint16(memCols * memRows) }

		// Enter edit mode
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			d.editMode = true
			d.editNibble = 0
			d.editBuf = hexByte(d.mem.Read(d.memCursor))
		}
	} else {
		d.handleMemEdit()
	}

	// Keep cursor in view
	if d.memCursor < d.memBase {
		d.memBase = d.memCursor &^ (memCols - 1)
	}
	if d.memCursor >= d.memBase+uint16(memCols*memRows) {
		d.memBase = (d.memCursor - uint16(memCols*(memRows-1))) &^ (memCols - 1)
	}

	// Mouse click
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		addr, ok := d.memPosToAddr(mx, my)
		if ok {
			d.memCursor = addr
			d.editMode = false
		}
	}
}

func (d *Debugger) handleMemEdit() {
	hexKeys := "0123456789ABCDEF"
	for _, ch := range hexKeys {
		key := ebiten.KeyA + ebiten.Key(ch-'A')
		if ch >= '0' && ch <= '9' {
			key = ebiten.Key0 + ebiten.Key(ch-'0')
		}
		if inpututil.IsKeyJustPressed(key) {
			cur := d.mem.Read(d.memCursor)
			var newVal byte
			nibble := byte(strings.IndexRune(hexKeys, ch))
			if d.editNibble == 0 {
				newVal = (nibble << 4) | (cur & 0x0F)
			} else {
				newVal = (cur & 0xF0) | nibble
			}
			d.mem.Write(d.memCursor, newVal)
			d.editNibble++
			if d.editNibble >= 2 {
				d.editNibble = 0
				d.memCursor++
				d.editMode = false
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.editMode = false
	}
}

func (d *Debugger) memPosToAddr(mx, my int) (uint16, bool) {
	startY := padding*2 + charH + 4 + charH + padding
	startX := padding + 5*charW // after "XXXX: "
	col := (mx - startX) / (3 * charW)
	row := (my - startY) / charH
	if col < 0 || col >= memCols || row < 0 || row >= memRows { return 0, false }
	return d.memBase + uint16(row*memCols+col), true
}

func (d *Debugger) drawMemory(screen *ebiten.Image, startY int) {
	y := startY

	// Header
	header := "      "
	for i := 0; i < memCols; i++ {
		header += fmt.Sprintf("%02X ", i)
	}
	header += "  ASCII"
	printAt(screen, header, padding, y, colDim)
	y += charH

	for row := 0; row < memRows; row++ {
		base := d.memBase + uint16(row*memCols)
		line := fmt.Sprintf("%04X: ", base)

		for col := 0; col < memCols; col++ {
			addr := base + uint16(col)
			b := d.mem.Read(addr)

			var c color.RGBA
			switch {
			case addr == d.memCursor && d.editMode:
				c = colEditing
			case addr == d.memCursor:
				c = colHighlight
			case d.memChanged[addr]:
				c = colChanged
			default:
				c = colHex
			}
			_ = c
			line += hexByte(b) + " "
		}

		// ASCII panel
		line += " "
		for col := 0; col < memCols; col++ {
			b := d.mem.Read(base + uint16(col))
			if b >= 0x20 && b < 0x7F {
				line += string(rune(b))
			} else {
				line += "."
			}
		}

		printAt(screen, line, padding, y, colText)

		// Highlight cursor row
		curRow := int(d.memCursor-d.memBase) / memCols
		if row == curRow {
			col := int(d.memCursor-d.memBase) % memCols
			hx := padding + (6+col*3)*charW
			drawRect(screen, hx-1, y-1, charW*2+2, charH+1, colTabActive)
			b := d.mem.Read(d.memCursor)
			txt := hexByte(b)
			if d.editMode { txt = d.editBuf }
			printAt(screen, txt, hx, y, colEditing)
		}

		y += charH
	}

	// Status bar
	b := d.mem.Read(d.memCursor)
	status := fmt.Sprintf("  Addr: $%04X  Val: $%02X (%d)  [Enter]=Edit  [Arrows]=Navigate  [PgUp/Dn]=Page",
		d.memCursor, b, b)
	if d.editMode {
		status = fmt.Sprintf("  EDIT $%04X — type hex nibbles  [Esc]=Cancel", d.memCursor)
	}
	printAt(screen, status, 0, WinH-charH-padding, colDim)
}
