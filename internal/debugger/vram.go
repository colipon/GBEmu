package debugger

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// VRAM page constants
const (
	VRAMPageTiles0 = iota // $8000-$87FF (sprites / BG signed)
	VRAMPageTiles1        // $8800-$97FF (full unsigned)
	VRAMPageBGMap0        // $9800 BG map
	VRAMPageBGMap1        // $9C00 BG map
	vramPageCount
)

var vramPageNames = [vramPageCount]string{
	"Tiles $8000 (bank0)",
	"Tiles $8800 (full)",
	"BG Map $9800",
	"BG Map $9C00",
}

// DMG shades for tile preview
var tileShades = [4]color.RGBA{
	{0xFF, 0xFF, 0xFF, 0xFF},
	{0xAA, 0xAA, 0xAA, 0xFF},
	{0x55, 0x55, 0x55, 0xFF},
	{0x00, 0x00, 0x00, 0xFF},
}

func (d *Debugger) updateVRAM() {
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		d.vramPage = (d.vramPage + 1) % vramPageCount
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		d.vramPage = (d.vramPage + vramPageCount - 1) % vramPageCount
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) {
		if d.vramZoom < 4 { d.vramZoom++ }
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) {
		if d.vramZoom > 1 { d.vramZoom-- }
	}
}

func (d *Debugger) drawVRAM(screen *ebiten.Image, startY int) {
	y := startY

	// Page selector
	for i, name := range vramPageNames {
		bg := colPanel
		if i == d.vramPage { bg = colTabActive }
		x := padding + i*(WinW/vramPageCount)
		drawRect(screen, x, y, WinW/vramPageCount-2, charH+2, bg)
		printAt(screen, name, x+2, y+1, colText)
	}
	y += charH + 6

	switch d.vramPage {
	case VRAMPageTiles0:
		d.drawTileGrid(screen, y, 0x0000, 128) // 128 tiles from $8000
	case VRAMPageTiles1:
		d.drawTileGrid(screen, y, 0x0000, 384) // all 384 tiles
	case VRAMPageBGMap0:
		d.drawBGMap(screen, y, 0x9800)
	case VRAMPageBGMap1:
		d.drawBGMap(screen, y, 0x9C00)
	}

	hint := fmt.Sprintf("  [←/→]=Page  [+/-]=Zoom x%d", d.vramZoom)
	printAt(screen, hint, 0, WinH-charH-padding, colDim)
}

// drawTileGrid renders up to maxTiles tiles starting at vramOffset
func (d *Debugger) drawTileGrid(screen *ebiten.Image, startY, vramOffset int, maxTiles int) {
	zoom := d.vramZoom
	tileSize := 8 * zoom
	tilesPerRow := (WinW - padding*2) / tileSize
	if tilesPerRow < 1 { tilesPerRow = 1 }

	hoverX, hoverY := ebiten.CursorPosition()

	for t := 0; t < maxTiles; t++ {
		tx := t % tilesPerRow
		ty := t / tilesPerRow
		px := padding + tx*tileSize
		py := startY + ty*(tileSize+1)

		if py+tileSize > WinH-charH*2 { break }

		// Decode tile
		tileImg := ebiten.NewImage(8, 8)
		pixels := make([]byte, 8*8*4)
		base := vramOffset + t*16
		for row := 0; row < 8; row++ {
			if base+row*2+1 >= len(d.mem.VRAM) { break }
			lo := d.mem.VRAM[base+row*2]
			hi := d.mem.VRAM[base+row*2+1]
			for col := 0; col < 8; col++ {
				bit := 7 - col
				idx := ((hi>>bit)&1)<<1 | ((lo >> bit) & 1)
				c := tileShades[idx]
				off := (row*8 + col) * 4
				pixels[off+0] = c.R; pixels[off+1] = c.G
				pixels[off+2] = c.B; pixels[off+3] = c.A
			}
		}
		tileImg.WritePixels(pixels)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(zoom), float64(zoom))
		op.GeoM.Translate(float64(px), float64(py))
		screen.DrawImage(tileImg, op)

		// Hover tooltip
		if hoverX >= px && hoverX < px+tileSize && hoverY >= py && hoverY < py+tileSize {
			drawRect(screen, px-1, py-1, tileSize+2, tileSize+2, colHighlight)
			screen.DrawImage(tileImg, op) // redraw on top
			addr := 0x8000 + vramOffset + t*16
			tip := fmt.Sprintf("Tile #%d  $%04X", t, addr)
			printAt(screen, tip, padding, WinH-charH*2-padding, colHighlight)
		}
	}
}

// drawBGMap renders the 32×32 tile map
func (d *Debugger) drawBGMap(screen *ebiten.Image, startY int, mapBase uint16) {
	zoom := d.vramZoom
	tileSize := 8 * zoom
	lcdc := d.mem.IO[0x40]
	signedTiles := lcdc&0x10 == 0

	hoverX, hoverY := ebiten.CursorPosition()
	var hoverInfo string

	for ty := 0; ty < 32; ty++ {
		for tx := 0; tx < 32; tx++ {
			px := padding + tx*tileSize
			py := startY + ty*tileSize
			if px+tileSize > WinW || py+tileSize > WinH-charH*2 { continue }

			mapIdx := mapBase - 0x8000 + uint16(ty*32+tx)
			tileIdx := d.mem.VRAM[mapIdx]

			var tileAddr uint16
			if signedTiles {
				tileAddr = uint16(0x9000-0x8000) + uint16(int16(int8(tileIdx))*16)
			} else {
				tileAddr = uint16(tileIdx) * 16
			}

			tileImg := ebiten.NewImage(8, 8)
			pixels := make([]byte, 8*8*4)
			for row := 0; row < 8; row++ {
				off := int(tileAddr) + row*2
				if off+1 >= len(d.mem.VRAM) { break }
				lo := d.mem.VRAM[off]
				hi := d.mem.VRAM[off+1]
				for col := 0; col < 8; col++ {
					bit := 7 - col
					idx := ((hi>>bit)&1)<<1 | ((lo >> bit) & 1)
					c := tileShades[idx]
					o := (row*8 + col) * 4
					pixels[o] = c.R; pixels[o+1] = c.G; pixels[o+2] = c.B; pixels[o+3] = c.A
				}
			}
			tileImg.WritePixels(pixels)

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(zoom), float64(zoom))
			op.GeoM.Translate(float64(px), float64(py))
			screen.DrawImage(tileImg, op)

			if hoverX >= px && hoverX < px+tileSize && hoverY >= py && hoverY < py+tileSize {
				drawRect(screen, px, py, tileSize, tileSize, color.RGBA{0xFF, 0x80, 0x80, 0x80})
				hoverInfo = fmt.Sprintf("Map(%d,%d) TileIdx=$%02X  Addr=$%04X", tx, ty, tileIdx, 0x8000+tileAddr)
			}
		}
	}

	// Viewport overlay (SCX/SCY)
	scx := int(d.mem.IO[0x43]) * zoom / 8
	scy := int(d.mem.IO[0x42]) * zoom / 8
	_ = scx; _ = scy

	if hoverInfo != "" {
		printAt(screen, hoverInfo, padding, WinH-charH*2-padding, colHighlight)
	}
}
