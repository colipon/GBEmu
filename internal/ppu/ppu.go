package ppu

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/colipon/gbemu/internal/mmu"
	"github.com/colipon/gbemu/internal/sgb"
)

const (
	ScreenWidth  = 160
	ScreenHeight = 144

	ModeHBlank = 0
	ModeVBlank = 1
	ModeOAM    = 2
	ModePixel  = 3
)

var dmgPalette = [4]color.RGBA{
	{0xE0, 0xF0, 0xD0, 0xFF},
	{0x88, 0xC0, 0x70, 0xFF},
	{0x34, 0x68, 0x56, 0xFF},
	{0x08, 0x18, 0x20, 0xFF},
}

type PPU struct {
	mem        *mmu.MMU
	SGB        *sgb.SGB
	pixels     []byte
	Screen     *ebiten.Image
	bgColorIdx [ScreenWidth]byte
	cycles     int
	mode       int
	ly         int
}

func New(mem *mmu.MMU, s *sgb.SGB) *PPU {
	p := &PPU{
		mem:    mem,
		SGB:    s,
		pixels: make([]byte, ScreenWidth*ScreenHeight*4),
		Screen: ebiten.NewImage(ScreenWidth, ScreenHeight),
	}
	mem.OnDMALaunch = func(src byte) { mem.DMA(src) }
	return p
}

func (p *PPU) Step(cycles int) {
	lcdc := p.mem.IO[0x40]
	if lcdc&0x80 == 0 {
		p.ly = 0; p.mem.IO[0x44] = 0
		p.setMode(ModeVBlank); p.cycles = 0
		return
	}
	p.cycles += cycles
	switch p.mode {
	case ModeOAM:
		if p.cycles >= 80 { p.cycles -= 80; p.setMode(ModePixel) }
	case ModePixel:
		if p.cycles >= 172 {
			p.cycles -= 172; p.renderScanline()
			p.setMode(ModeHBlank); p.checkLYC()
		}
	case ModeHBlank:
		if p.cycles >= 204 {
			p.cycles -= 204; p.ly++; p.mem.IO[0x44] = byte(p.ly)
			if p.ly == 144 {
				p.setMode(ModeVBlank)
				p.mem.RequestInterrupt(mmu.IntVBlank)
				p.flush()
			} else {
				p.setMode(ModeOAM)
			}
			p.checkLYC()
		}
	case ModeVBlank:
		if p.cycles >= 456 {
			p.cycles -= 456; p.ly++
			if p.ly > 153 { p.ly = 0; p.setMode(ModeOAM) }
			p.mem.IO[0x44] = byte(p.ly); p.checkLYC()
		}
	}
}

func (p *PPU) setMode(m int) {
	p.mode = m
	stat := (p.mem.IO[0x41] & 0xFC) | byte(m)
	fire := false
	switch m {
	case ModeHBlank: fire = stat&0x08 != 0
	case ModeVBlank: fire = stat&0x10 != 0
	case ModeOAM:    fire = stat&0x20 != 0
	}
	if fire { p.mem.RequestInterrupt(mmu.IntLCDStat) }
	p.mem.IO[0x41] = stat
}

func (p *PPU) checkLYC() {
	lyc := p.mem.IO[0x45]; stat := p.mem.IO[0x41]
	if byte(p.ly) == lyc {
		stat |= 0x04
		if stat&0x40 != 0 { p.mem.RequestInterrupt(mmu.IntLCDStat) }
	} else {
		stat &^= 0x04
	}
	p.mem.IO[0x41] = stat
}

func (p *PPU) renderScanline() {
	lcdc := p.mem.IO[0x40]
	if lcdc&0x01 != 0 { p.renderBG() }
	if lcdc&0x20 != 0 { p.renderWindow() }
	if lcdc&0x02 != 0 { p.renderSprites() }
}

func (p *PPU) mapColor(colorIdx byte, bgp byte, tileX, tileY int) color.RGBA {
	shade := (bgp >> (colorIdx * 2)) & 0x03
	if p.SGB != nil && p.SGB.Enabled && p.SGB.MaskMode == 0 {
		return p.SGB.ColorForIndex(colorIdx, tileX, tileY)
	}
	return dmgPalette[shade]
}

func (p *PPU) renderBG() {
	lcdc := p.mem.IO[0x40]; scy := p.mem.IO[0x42]; scx := p.mem.IO[0x43]; bgp := p.mem.IO[0x47]
	tileMapBase := uint16(0x9800)
	if lcdc&0x08 != 0 { tileMapBase = 0x9C00 }
	signedTiles := lcdc&0x10 == 0
	ly := byte(p.ly); y := ly + scy
	for lx := byte(0); lx < 160; lx++ {
		x := lx + scx
		tileCol := int(x / 8); tileRow := int(y / 8)
		mapIdx := tileMapBase - 0x8000 + uint16(tileRow)*32 + uint16(tileCol)
		tileIdx := p.mem.VRAM[mapIdx]
		var tileAddr uint16
		if signedTiles {
			tileAddr = uint16(0x9000) + uint16(int16(int8(tileIdx))*16)
		} else {
			tileAddr = 0x8000 + uint16(tileIdx)*16
		}
		off := tileAddr - 0x8000 + uint16((y%8)*2)
		lo := p.mem.VRAM[off]; hi := p.mem.VRAM[off+1]
		bit := 7 - (x % 8)
		colorIdx := ((hi>>bit)&1)<<1 | ((lo >> bit) & 1)
		p.bgColorIdx[lx] = colorIdx
		p.setPixel(int(lx), p.ly, p.mapColor(colorIdx, bgp, tileCol, tileRow))
	}
}

func (p *PPU) renderWindow() {
	lcdc := p.mem.IO[0x40]; wy := p.mem.IO[0x4A]; wx := int(p.mem.IO[0x4B]) - 7; bgp := p.mem.IO[0x47]
	if p.ly < int(wy) { return }
	tileMapBase := uint16(0x9800)
	if lcdc&0x40 != 0 { tileMapBase = 0x9C00 }
	signedTiles := lcdc&0x10 == 0
	windowY := p.ly - int(wy)
	for lx := 0; lx < 160; lx++ {
		if lx < wx { continue }
		windowX := lx - wx
		tileCol := windowX / 8; tileRow := windowY / 8
		mapIdx := tileMapBase - 0x8000 + uint16(tileRow)*32 + uint16(tileCol)
		tileIdx := p.mem.VRAM[mapIdx]
		var tileAddr uint16
		if signedTiles {
			tileAddr = uint16(0x9000) + uint16(int16(int8(tileIdx))*16)
		} else {
			tileAddr = 0x8000 + uint16(tileIdx)*16
		}
		off := tileAddr - 0x8000 + uint16((windowY%8)*2)
		lo := p.mem.VRAM[off]; hi := p.mem.VRAM[off+1]
		bit := 7 - (windowX % 8)
		colorIdx := ((hi>>bit)&1)<<1 | ((lo >> bit) & 1)
		p.bgColorIdx[lx] = colorIdx
		p.setPixel(lx, p.ly, p.mapColor(colorIdx, bgp, tileCol, tileRow))
	}
}

func (p *PPU) renderSprites() {
	lcdc := p.mem.IO[0x40]; obp0 := p.mem.IO[0x48]; obp1 := p.mem.IO[0x49]
	spriteHeight := 8
	if lcdc&0x04 != 0 { spriteHeight = 16 }
	count := 0
	for i := 0; i < 40 && count < 10; i++ {
		base := i * 4
		sprY := int(p.mem.OAM[base]) - 16; sprX := int(p.mem.OAM[base+1]) - 8
		tileNum := p.mem.OAM[base+2]; attrs := p.mem.OAM[base+3]
		if p.ly < sprY || p.ly >= sprY+spriteHeight { continue }
		count++
		palette := obp0
		if attrs&0x10 != 0 { palette = obp1 }
		flipX := attrs&0x20 != 0; flipY := attrs&0x40 != 0; bgPri := attrs&0x80 != 0
		row := p.ly - sprY
		if flipY { row = spriteHeight - 1 - row }
		if spriteHeight == 16 { tileNum &^= 0x01 }
		tileAddr := uint16(tileNum)*16 + uint16(row)*2
		lo := p.mem.VRAM[tileAddr]; hi := p.mem.VRAM[tileAddr+1]
		for px := 0; px < 8; px++ {
			sx := sprX + px
			if sx < 0 || sx >= 160 { continue }
			bit := 7 - px
			if flipX { bit = px }
			colorIdx := ((hi>>bit)&1)<<1 | ((lo >> bit) & 1)
			if colorIdx == 0 { continue }
			if bgPri && p.bgColorIdx[sx] != 0 { continue }
			shade := (palette >> (colorIdx * 2)) & 0x03
			var rgba color.RGBA
			if p.SGB != nil && p.SGB.Enabled {
				rgba = p.SGB.Palettes[0][shade]
			} else {
				rgba = dmgPalette[shade]
			}
			p.setPixel(sx, p.ly, rgba)
		}
	}
}

func (p *PPU) setPixel(x, y int, c color.RGBA) {
	off := (y*ScreenWidth + x) * 4
	p.pixels[off+0] = c.R; p.pixels[off+1] = c.G
	p.pixels[off+2] = c.B; p.pixels[off+3] = c.A
}

func (p *PPU) flush() { p.Screen.WritePixels(p.pixels) }
