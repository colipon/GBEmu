package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/colipon/gbemu/internal/cartridge"
	"github.com/colipon/gbemu/internal/cpu"
	"github.com/colipon/gbemu/internal/debugger"
	"github.com/colipon/gbemu/internal/joypad"
	"github.com/colipon/gbemu/internal/mmu"
	"github.com/colipon/gbemu/internal/ppu"
	"github.com/colipon/gbemu/internal/sgb"
	"github.com/colipon/gbemu/internal/timer"
)

const (
	scale          = 3
	cpuHz          = 4194304
	frameHz        = 60
	cyclesPerFrame = cpuHz / frameHz
)

// Layout modes
const (
	layoutGame  = iota // game only
	layoutDebug        // game + debugger side by side
)

func gameSize(sgbMode bool) (int, int) {
	if sgbMode {
		return sgb.SGBBorderW * scale, sgb.SGBBorderH * scale
	}
	return ppu.ScreenWidth * scale, ppu.ScreenHeight * scale
}

// Game holds all emulator state
type Game struct {
	cpu    *cpu.CPU
	mem    *mmu.MMU
	ppu    *ppu.PPU
	timer  *timer.Timer
	joypad *joypad.Joypad
	sgb    *sgb.SGB
	dbg    *debugger.Debugger

	border      *ebiten.Image
	borderDirty bool

	title   string
	sgbMode bool
	layout  int
}

func NewGame(romPath string, sgbMode bool) (*Game, error) {
	cart, err := cartridge.Load(romPath)
	if err != nil {
		return nil, err
	}

	mem := mmu.New(cart)

	var s *sgb.SGB
	if sgbMode {
		s = sgb.New()
	}

	c := cpu.New(mem)
	p := ppu.New(mem, s)
	t := timer.New(mem)
	j := joypad.New(mem, s)
	dbg := debugger.New(mem, c)

	g := &Game{
		cpu:     c,
		mem:     mem,
		ppu:     p,
		timer:   t,
		joypad:  j,
		sgb:     s,
		dbg:     dbg,
		title:   cart.Title(),
		sgbMode: sgbMode,
		layout:  layoutGame,
	}

	if sgbMode {
		g.border = ebiten.NewImage(sgb.SGBBorderW, sgb.SGBBorderH)
		g.borderDirty = true
		origPPUWrite := mem.OnPPUWrite
		mem.OnPPUWrite = func(addr uint16, val byte) {
			if origPPUWrite != nil {
				origPPUWrite(addr, val)
			}
			if addr == 0xFF40 {
				g.borderDirty = true
			}
		}
	}

	return g, nil
}

func (g *Game) Update() error {
	// Toggle debugger
	if inpututil.IsKeyJustPressed(ebiten.KeyD) && !g.dbg.IsPaused() || 
	   inpututil.IsKeyJustPressed(ebiten.KeyD) {
		if g.layout == layoutGame {
			g.layout = layoutDebug
		} else {
			g.layout = layoutGame
		}
		g.updateWindowSize()
	}

	// Debugger update (handles its own input)
	if g.layout == layoutDebug {
		g.dbg.Update()
	}

	// Check breakpoints
	g.dbg.CheckBreakpoint(g.cpu.PC)

	// Emulation
	if !g.dbg.IsPaused() || g.dbg.ConsumeStep() {
		remaining := cyclesPerFrame
		for remaining > 0 {
			g.joypad.Update()
			cyc := g.cpu.Step()
			g.ppu.Step(cyc)
			g.timer.Step(cyc)
			remaining -= cyc
			// Re-check breakpoints mid-frame
			g.dbg.CheckBreakpoint(g.cpu.PC)
			if g.dbg.IsPaused() {
				break
			}
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	gw, _ := gameSize(g.sgbMode)

	// ── Game area ────────────────────────────────────────────────────────────
	if g.sgbMode && g.sgb != nil {
		if g.borderDirty {
			g.sgb.RenderBorder(g.border)
			g.borderDirty = false
		}
		borderOp := &ebiten.DrawImageOptions{}
		borderOp.GeoM.Scale(scale, scale)
		screen.DrawImage(g.border, borderOp)

		if g.sgb.MaskMode != 2 {
			gbOp := &ebiten.DrawImageOptions{}
			gbOp.GeoM.Scale(scale, scale)
			gbOp.GeoM.Translate(float64(sgb.GBOffsetX*scale), float64(sgb.GBOffsetY*scale))
			screen.DrawImage(g.ppu.Screen, gbOp)
		}
	} else {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(g.ppu.Screen, op)
	}

	// ── Debugger panel ───────────────────────────────────────────────────────
	if g.layout == layoutDebug {
		dbgImg := ebiten.NewImage(debugger.WinW, debugger.WinH)
		g.dbg.Draw(dbgImg)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(gw), 0)
		screen.DrawImage(dbgImg, op)
	}

	// HUD
	mode := "DMG"
	if g.sgbMode { mode = "SGB" }
	pauseStr := ""
	if g.dbg.IsPaused() { pauseStr = "  ⏸ PAUSED" }
	ebitenutil.DebugPrintAt(screen,
		fmt.Sprintf("[%s] %s  FPS:%.0f%s  [D]=Debugger", mode, g.title, ebiten.ActualFPS(), pauseStr),
		4, 4)
}

func (g *Game) Layout(_, _ int) (int, int) {
	gw, gh := gameSize(g.sgbMode)
	if g.layout == layoutDebug {
		return gw + debugger.WinW, max(gh, debugger.WinH)
	}
	return gw, gh
}

func (g *Game) updateWindowSize() {
	gw, gh := gameSize(g.sgbMode)
	if g.layout == layoutDebug {
		ebiten.SetWindowSize(gw+debugger.WinW, max(gh, debugger.WinH))
	} else {
		ebiten.SetWindowSize(gw, gh)
	}
}

func max(a, b int) int {
	if a > b { return a }
	return b
}

func main() {
	sgbMode := flag.Bool("sgb", true, "Enable Super Game Boy mode")
	dmgMode := flag.Bool("dmg", false, "Force DMG mode")
	dbgMode := flag.Bool("debug", false, "Open debugger on startup")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: gbemu [--sgb|--dmg] [--debug] <rom.gb>")
		os.Exit(1)
	}

	useSGB := *sgbMode && !*dmgMode

	game, err := NewGame(args[0], useSGB)
	if err != nil {
		log.Fatal(err)
	}

	if *dbgMode {
		game.layout = layoutDebug
	}

	gw, gh := gameSize(useSGB)
	w, h := gw, gh
	if game.layout == layoutDebug {
		w = gw + debugger.WinW
		h = max(gh, debugger.WinH)
	}

	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	modeLabel := "SGB"
	if !useSGB { modeLabel = "DMG" }
	ebiten.SetWindowTitle(fmt.Sprintf("gbemu [%s] — %s", modeLabel, game.title))
	ebiten.SetTPS(frameHz)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
