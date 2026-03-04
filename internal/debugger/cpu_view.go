package debugger

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (d *Debugger) updateCPU() {
	// Space = pause/resume
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		d.paused = !d.paused
	}
	// N = step one instruction (only when paused)
	if d.paused && inpututil.IsKeyJustPressed(ebiten.KeyN) {
		d.stepOnce = true
	}
	// B = toggle breakpoint at current PC
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		pc := d.cpu.PC
		if d.breakpoints[pc] {
			delete(d.breakpoints, pc)
		} else {
			d.breakpoints[pc] = true
		}
	}
	// Delete = clear all breakpoints
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		d.breakpoints = make(map[uint16]bool)
	}
}

func (d *Debugger) drawCPU(screen *ebiten.Image, startY int) {
	y := startY
	col2 := WinW/2 + padding

	// ── Registers ─────────────────────────────────────────────────────────────
	printAt(screen, "─── Registers ───", padding, y, colDim)
	y += charH

	regs := []struct{ name, val string }{
		{"AF", fmt.Sprintf("$%04X  (A=$%02X  F=$%02X)", d.cpu.AF(), d.cpu.A, d.cpu.F)},
		{"BC", fmt.Sprintf("$%04X  (B=$%02X  C=$%02X)", d.cpu.BC(), d.cpu.B, d.cpu.C)},
		{"DE", fmt.Sprintf("$%04X  (D=$%02X  E=$%02X)", d.cpu.DE(), d.cpu.D, d.cpu.E)},
		{"HL", fmt.Sprintf("$%04X  (H=$%02X  L=$%02X)", d.cpu.HL(), d.cpu.H, d.cpu.L)},
		{"SP", fmt.Sprintf("$%04X", d.cpu.SP)},
		{"PC", fmt.Sprintf("$%04X", d.cpu.PC)},
	}
	for _, r := range regs {
		line := fmt.Sprintf("  %-3s = %s", r.name, r.val)
		printAt(screen, line, padding, y, colHex)
		y += charH
	}

	// ── Flags ─────────────────────────────────────────────────────────────────
	y += 4
	printAt(screen, "─── Flags ───", padding, y, colDim)
	y += charH

	flagStr := func(bit int, name string) string {
		if d.cpu.F>>bit&1 == 1 {
			return name
		}
		return strings.ToLower(name)
	}
	flags := fmt.Sprintf("  Z=%s  N=%s  H=%s  C=%s",
		flagStr(7, "Z"), flagStr(6, "N"), flagStr(5, "H"), flagStr(4, "C"))
	printAt(screen, flags, padding, y, colText)
	y += charH

	// ── IME / Halted ──────────────────────────────────────────────────────────
	y += 4
	ime := "OFF"
	if d.cpu.IME { ime = "ON" }
	halted := ""
	if d.cpu.Halted { halted = "  HALTED" }
	printAt(screen, fmt.Sprintf("  IME=%s  IE=$%02X  IF=$%02X%s",
		ime, d.mem.IE, d.mem.IF, halted), padding, y, colText)
	y += charH

	// ── IO Registers ──────────────────────────────────────────────────────────
	y += 8
	printAt(screen, "─── IO Registers ───", padding, y, colDim)
	y += charH

	ioRegs := []struct{ name string; addr uint16 }{
		{"LCDC", 0xFF40}, {"STAT", 0xFF41}, {"SCY", 0xFF42}, {"SCX", 0xFF43},
		{"LY",   0xFF44}, {"LYC",  0xFF45}, {"BGP", 0xFF47}, {"OBP0", 0xFF48},
		{"OBP1", 0xFF49}, {"WY",   0xFF4A}, {"WX",  0xFF4B},
		{"DIV",  0xFF04}, {"TIMA", 0xFF05}, {"TMA", 0xFF06}, {"TAC",  0xFF07},
	}
	for i, r := range ioRegs {
		col := padding
		if i >= 8 { col = col2 }
		row := i % 8
		printAt(screen, fmt.Sprintf("  %-5s $%02X", r.name, d.mem.Read(r.addr)),
			col, startY+charH*2+4 + row*charH, colText)
	}

	// ── Execution control ─────────────────────────────────────────────────────
	ctrlY := WinH/2 + 20
	printAt(screen, "─── Execution ───", padding, ctrlY, colDim)
	ctrlY += charH

	status := "▶ RUNNING"
	statusCol := colAddr
	if d.paused {
		status = "⏸ PAUSED"
		statusCol = colHighlight
	}
	printAt(screen, fmt.Sprintf("  %s", status), padding, ctrlY, statusCol)
	ctrlY += charH

	controls := []string{
		"[Space] Pause / Resume",
		"[N]     Step one instruction (when paused)",
		"[B]     Toggle breakpoint at PC",
		"[Del]   Clear all breakpoints",
	}
	for _, c := range controls {
		printAt(screen, "  "+c, padding, ctrlY, colDim)
		ctrlY += charH
	}

	// ── Breakpoints ───────────────────────────────────────────────────────────
	ctrlY += 8
	printAt(screen, fmt.Sprintf("─── Breakpoints (%d) ───", len(d.breakpoints)), padding, ctrlY, colDim)
	ctrlY += charH

	if len(d.breakpoints) == 0 {
		printAt(screen, "  (none)", padding, ctrlY, colDim)
	} else {
		count := 0
		for addr := range d.breakpoints {
			if count >= 8 { break }
			active := ""
			if addr == d.cpu.PC { active = " ◀ PC" }
			printAt(screen, fmt.Sprintf("  $%04X%s", addr, active), padding+count/4*(WinW/2), ctrlY+(count%4)*charH, colBreak)
			count++
		}
	}

	// Stack peek
	stackY := ctrlY
	printAt(screen, "─── Stack (SP) ───", col2, stackY, colDim)
	stackY += charH
	for i := 0; i < 8; i++ {
		addr := d.cpu.SP + uint16(i*2)
		val := d.mem.Read16(addr)
		marker := "  "
		if i == 0 { marker = "→ " }
		printAt(screen, fmt.Sprintf("%s$%04X: $%04X", marker, addr, val), col2, stackY, colHex)
		stackY += charH
	}
}
