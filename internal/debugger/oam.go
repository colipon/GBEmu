package debugger

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (d *Debugger) updateOAM() {
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		if d.oamSelected < 39 { d.oamSelected++ }
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		if d.oamSelected > 0 { d.oamSelected-- }
	}
}

func (d *Debugger) drawOAM(screen *ebiten.Image, startY int) {
	y := startY

	// Header
	header := fmt.Sprintf("  %-4s  %-5s  %-5s  %-4s  %-4s  %-5s  %-5s  %-5s  %-5s",
		"#", "Y", "X", "Tile", "Attr", "Pal", "XFlip", "YFlip", "BGPri")
	printAt(screen, header, padding, y, colDim)
	y += charH

	lcdc := d.mem.IO[0x40]
	sprH := 8
	if lcdc&0x04 != 0 { sprH = 16 }

	for i := 0; i < 40; i++ {
		base := i * 4
		sprY := int(d.mem.OAM[base]) - 16
		sprX := int(d.mem.OAM[base+1]) - 8
		tile := d.mem.OAM[base+2]
		attr := d.mem.OAM[base+3]

		pal := "OBP0"
		if attr&0x10 != 0 { pal = "OBP1" }
		xFlip := attr&0x20 != 0
		yFlip := attr&0x40 != 0
		bgPri := attr&0x80 != 0

		visible := sprY >= -sprH && sprY < 144 && sprX >= -8 && sprX < 160

		bg := colPanel
		if i == d.oamSelected { bg = colTabActive }
		if !visible { bg = colBG }

		drawRect(screen, padding, y-1, WinW-padding*2, charH+1, bg)

		line := fmt.Sprintf("  %-4d  %-5d  %-5d  $%-3X  $%-3X  %-5s  %-5v  %-5v  %-5v",
			i, sprY, sprX, tile, attr, pal, xFlip, yFlip, bgPri)

		tc := colText
		if !visible { tc = colDim }
		if i == d.oamSelected { tc = colHighlight }
		printAt(screen, line, padding, y, tc)

		// Mini sprite preview for selected
		if i == d.oamSelected {
			d.drawSpritePreview(screen, WinW-padding-64, startY+4, tile, attr, sprH)
		}

		y += charH
		if y > WinH-charH*3 { break }
	}

	hint := fmt.Sprintf("  [↑/↓]=Select  Sprite size: %dpx  OBP0=$%02X  OBP1=$%02X",
		sprH, d.mem.IO[0x48], d.mem.IO[0x49])
	printAt(screen, hint, 0, WinH-charH-padding, colDim)
}

func (d *Debugger) drawSpritePreview(screen *ebiten.Image, x, y, tileNum int, attr byte, sprH int) {
	zoom := 4
	flipX := attr&0x20 != 0
	flipY := attr&0x40 != 0
	pal := d.mem.IO[0x48]
	if attr&0x10 != 0 { pal = d.mem.IO[0x49] }

	tileN := byte(tileNum)
	if sprH == 16 { tileN &^= 0x01 }

	img := ebiten.NewImage(8, sprH)
	pixels := make([]byte, 8*sprH*4)

	for row := 0; row < sprH; row++ {
		srcRow := row
		if flipY { srcRow = sprH - 1 - row }
		tileAddr := uint16(tileN)*16 + uint16(srcRow)*2
		if int(tileAddr)+1 >= len(d.mem.VRAM) { continue }
		lo := d.mem.VRAM[tileAddr]
		hi := d.mem.VRAM[tileAddr+1]
		for col := 0; col < 8; col++ {
			srcCol := col
			if flipX { srcCol = 7 - col }
			bit := 7 - srcCol
			colorIdx := ((hi>>bit)&1)<<1 | ((lo >> bit) & 1)
			shade := (pal >> (colorIdx * 2)) & 0x03
			c := tileShades[shade]
			if colorIdx == 0 { c.A = 0x40 } // transparent
			off := (row*8 + col) * 4
			pixels[off+0] = c.R; pixels[off+1] = c.G; pixels[off+2] = c.B; pixels[off+3] = c.A
		}
	}
	img.WritePixels(pixels)

	// Border
	drawRect(screen, x-2, y-2, 8*zoom+4, sprH*zoom+4, colDim)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(zoom), float64(zoom))
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(img, op)
}
