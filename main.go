package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/user/gbemu/internal/cartridge"
	"github.com/user/gbemu/internal/cpu"
	"github.com/user/gbemu/internal/joypad"
	"github.com/user/gbemu/internal/mmu"
	"github.com/user/gbemu/internal/ppu"
	"github.com/user/gbemu/internal/sgb"
	"github.com/user/gbemu/internal/timer"
)

const (
	scale          = 3
	cpuHz          = 4194304
	frameHz        = 60
	cyclesPerFrame = cpuHz / frameHz
)

// windowWidth / windowHeight depend on mode
func windowSize(sgbMode bool) (int, int) {
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
	sgb    *sgb.SGB // nil in DMG mode

	border      *ebiten.Image // 256×224 SGB border composite
	borderDirty bool

	title   string
	sgbMode bool
}

// NewGame initialises all components
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

	g := &Game{
		cpu:     c,
		mem:     mem,
		ppu:     p,
		timer:   t,
		joypad:  j,
		sgb:     s,
		title:   cart.Title(),
		sgbMode: sgbMode,
	}

	if sgbMode {
		g.border = ebiten.NewImage(sgb.SGBBorderW, sgb.SGBBorderH)
		g.borderDirty = true

		// Watch for CHR_TRN / PCT_TRN to trigger border re-render
		origPPUWrite := mem.OnPPUWrite
		mem.OnPPUWrite = func(addr uint16, val byte) {
			if origPPUWrite != nil {
				origPPUWrite(addr, val)
			}
			// CHR_TRN (0x13) and PCT_TRN (0x14) commands transfer data
			// via VRAM; we mark border dirty after any LCDC write
			if addr == 0xFF40 {
				g.borderDirty = true
			}
		}
	}

	return g, nil
}

func (g *Game) Update() error {
	g.joypad.Update()

	remaining := cyclesPerFrame
	for remaining > 0 {
		cyc := g.cpu.Step()
		g.ppu.Step(cyc)
		g.timer.Step(cyc)
		remaining -= cyc
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	if g.sgbMode && g.sgb != nil {
		// Re-render border when flagged
		if g.borderDirty {
			g.sgb.RenderBorder(g.border)
			g.borderDirty = false
		}

		// Draw border (full 256×224 scaled)
		borderOp := &ebiten.DrawImageOptions{}
		borderOp.GeoM.Scale(scale, scale)
		screen.DrawImage(g.border, borderOp)

		// Draw GB screen inset at (GBOffsetX, GBOffsetY) in border coords
		gbOp := &ebiten.DrawImageOptions{}
		gbOp.GeoM.Scale(scale, scale)
		gbOp.GeoM.Translate(float64(sgb.GBOffsetX*scale), float64(sgb.GBOffsetY*scale))

		// Apply mask mode
		if g.sgb.MaskMode == 2 {
			// Black mask — don't draw GB screen
		} else {
			screen.DrawImage(g.ppu.Screen, gbOp)
		}
	} else {
		// DMG mode: plain 160×144 scaled
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale, scale)
		screen.DrawImage(g.ppu.Screen, op)
	}

	// HUD
	mode := "DMG"
	if g.sgbMode {
		mode = "SGB"
	}
	ebitenutil.DebugPrintAt(screen,
		fmt.Sprintf("[%s] %s  FPS: %.0f", mode, g.title, ebiten.ActualFPS()),
		4, 4)
}

func (g *Game) Layout(_, _ int) (int, int) {
	return windowSize(g.sgbMode)
}

func main() {
	sgbMode := flag.Bool("sgb", true, "Enable Super Game Boy mode (default: true)")
	dmgMode := flag.Bool("dmg", false, "Force DMG (original Game Boy) mode")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: gbemu [--sgb|--dmg] <rom.gb>")
		os.Exit(1)
	}

	// --dmg overrides --sgb
	useSGB := *sgbMode && !*dmgMode

	game, err := NewGame(args[0], useSGB)
	if err != nil {
		log.Fatal(err)
	}

	w, h := windowSize(useSGB)
	ebiten.SetWindowSize(w, h)

	modeLabel := "SGB"
	if !useSGB {
		modeLabel = "DMG"
	}
	ebiten.SetWindowTitle(fmt.Sprintf("gbemu [%s] — %s", modeLabel, game.title))
	ebiten.SetTPS(frameHz)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
