package sgb

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// BorderTileSize is the pixel size of one SNES border tile
const BorderTileSize = 8

// RenderBorder draws the 256×224 SGB border onto dst.
// The 160×144 GB screen area (offset 48,40) is left transparent.
func (s *SGB) RenderBorder(dst *ebiten.Image) {
	if !s.BorderReady {
		dst.Fill(color.RGBA{0x18, 0x18, 0x18, 0xFF}) // dark default
		return
	}

	// Border tile map layout (from PCT_TRN):
	//   0x000–0x7FF : tile map  (32×28 entries, each 2 bytes)
	//   0x800–0xFFF : palette data (16 palettes × 16 colors × 2 bytes BGR555)
	//
	// Tile map entry bits:
	//   15    : Y-flip
	//   14    : X-flip
	//   13–10 : palette (0–7, offset from SNES OBJ palette base)
	//   9     : priority
	//   8     : VRAM bank (ignored — we only have 1 CHR bank)
	//   7–0   : tile number

	// Parse the 8 border palettes (each 16 colors) from offset 0x800
	type pal16 [16]color.RGBA
	var palettes [8]pal16
	for p := 0; p < 8; p++ {
		base := 0x800 + p*32
		for c := 0; c < 16; c++ {
			lo := s.BorderMap[base+c*2]
			hi := s.BorderMap[base+c*2+1]
			palettes[p][c] = bgr555ToRGBA(lo, hi)
		}
	}

	// Render 32×28 tiles
	pixels := make([]byte, SGBBorderW*SGBBorderH*4)

	for ty := 0; ty < 28; ty++ {
		for tx := 0; tx < 32; tx++ {
			mapOff := (ty*32 + tx) * 2
			lo := s.BorderMap[mapOff]
			hi := s.BorderMap[mapOff+1]
			entry := uint16(lo) | uint16(hi)<<8

			tileNum := int(entry & 0xFF)
			// bank      := (entry >> 8) & 1   // unused
			palIdx := int((entry >> 10) & 0x07)
			xFlip := entry&0x4000 != 0
			yFlip := entry&0x8000 != 0

			for py := 0; py < 8; py++ {
				srcY := py
				if yFlip {
					srcY = 7 - py
				}
				// 4bpp SNES tile: each row = 4 bytes (2 bitplanes × 2 bytes)
				// CHR data: plane0 row0 lo, plane0 row0 hi, plane1 row0 lo, …
				// SNES 4bpp layout: bp0lo bp0hi bp1lo bp1hi  (per row)
				rowBase := tileNum*32 + srcY*2
				if rowBase+1 >= len(s.BorderTiles) {
					continue
				}
				bp0lo := s.BorderTiles[rowBase]
				bp0hi := s.BorderTiles[rowBase+1]
				bp1lo := s.BorderTiles[rowBase+16]
				bp1hi := s.BorderTiles[rowBase+17]

				for px := 0; px < 8; px++ {
					srcX := px
					if xFlip {
						srcX = 7 - px
					}
					bit := 7 - srcX
					c0 := (bp0lo >> bit) & 1
					c1 := (bp0hi >> bit) & 1
					c2 := (bp1lo >> bit) & 1
					c3 := (bp1hi >> bit) & 1
					colorIdx := c0 | c1<<1 | c2<<2 | c3<<3

					// Color 0 of palette 0 = transparent (show black background)
					rgba := palettes[palIdx][colorIdx]
					if colorIdx == 0 {
						rgba = color.RGBA{0, 0, 0, 0xFF}
					}

					screenX := tx*8 + px
					screenY := ty*8 + py

					// Skip the GB viewport area — leave it for the PPU
					if screenX >= GBOffsetX && screenX < GBOffsetX+160 &&
						screenY >= GBOffsetY && screenY < GBOffsetY+144 {
						continue
					}

					off := (screenY*SGBBorderW + screenX) * 4
					if off+3 < len(pixels) {
						pixels[off+0] = rgba.R
						pixels[off+1] = rgba.G
						pixels[off+2] = rgba.B
						pixels[off+3] = rgba.A
					}
				}
			}
		}
	}

	dst.WritePixels(pixels)
}
