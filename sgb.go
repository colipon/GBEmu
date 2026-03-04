// Package sgb implements Super Game Boy (SGB) support.
//
// The GB communicates with the SNES-side SGB chip by sending
// 1-bit serial packets over the Joypad register (FF00).
// Each packet is 128 bits (16 bytes): a 1-byte command/length header
// followed by 15 bytes of payload, terminated by a stop bit.
//
// Reference: https://gbdev.io/pandocs/SGB_Functions.html
package sgb

import (
	"image/color"
)

// Screen geometry
const (
	SGBBorderW = 256
	SGBBorderH = 224
	GBOffsetX  = 48
	GBOffsetY  = 40
)

// SGB command codes
const (
	CmdPALxx    = 0x00 // PAL01..PAL23 — set color palettes
	CmdPAL01    = 0x00
	CmdPAL23    = 0x01
	CmdPAL03    = 0x02
	CmdPAL12    = 0x03
	CmdATF_SET  = 0x04
	CmdMASK_EN  = 0x17
	CmdPAL_SET  = 0x0A
	CmdPAL_TRN  = 0x0B
	CmdCHR_TRN  = 0x13
	CmdPCT_TRN  = 0x14
	CmdATRN     = 0x15
	CmdOBJ_TRN  = 0x18
	CmdMLT_REQ  = 0x11
)

// Palette holds 4 15-bit BGR555 colors converted to RGBA
type Palette [4]color.RGBA

// ATF (Attribute File) maps each 20×18 tile to one of 4 palettes
type ATF [20 * 18]byte

// SGB holds all Super Game Boy state
type SGB struct {
	Enabled bool

	// 4 system palettes (each 4 colors)
	Palettes [4]Palette

	// Current ATF (which palette each BG tile uses)
	ATF ATF

	// Border tile/map data (transferred via PCT_TRN / CHR_TRN)
	BorderTiles [0x2000]byte // 8KB CHR data
	BorderMap   [0x1000]byte // tile map + attributes
	BorderReady bool

	// System palette RAM (transferred via PAL_TRN) — 512 palettes × 4 colors
	SysPalRAM [512 * 4 * 2]byte // raw 15-bit pairs

	// Mask mode: 0=none 1=freeze 2=black 3=color0
	MaskMode byte

	// Packet reception state
	bitbuf  uint64
	bitbuf2 uint64
	bitcnt  int
	prevP1  byte
	packets [7][16]byte // up to 7 packets per transfer
	pktcnt  int
	pktlen  int // expected packet count for current command
}

// New returns an initialised SGB with default DMG-green palette
func New() *SGB {
	s := &SGB{Enabled: true}
	// Default palette: classic DMG green tones as SGB palette 0
	s.Palettes[0] = Palette{
		{0xE0, 0xF0, 0xD0, 0xFF},
		{0x88, 0xC0, 0x70, 0xFF},
		{0x34, 0x68, 0x56, 0xFF},
		{0x08, 0x18, 0x20, 0xFF},
	}
	for i := 1; i < 4; i++ {
		s.Palettes[i] = s.Palettes[0]
	}
	return s
}

// UpdateJoypad is called every time FF00 is written.
// It detects the SGB serial protocol pulse train and accumulates packets.
func (s *SGB) UpdateJoypad(val byte) {
	// Reset pulse: P14=0 P15=0
	if val&0x30 == 0x00 {
		if s.bitcnt > 0 && s.pktcnt < s.pktlen {
			// incomplete — discard
		}
		s.bitcnt = 0
		s.bitbuf = 0
		s.bitbuf2 = 0
		s.prevP1 = val
		return
	}

	prev := s.prevP1
	s.prevP1 = val

	// Falling edge on P15 (strobe) with P14 high = data bit
	if prev&0x20 != 0 && val&0x20 == 0 {
		bit := byte(0)
		if val&0x10 != 0 {
			bit = 1
		}
		if s.bitcnt < 64 {
			s.bitbuf |= uint64(bit) << s.bitcnt
		} else {
			s.bitbuf2 |= uint64(bit) << (s.bitcnt - 64)
		}
		s.bitcnt++

		if s.bitcnt == 128 {
			s.receivePacket()
			s.bitcnt = 0
			s.bitbuf = 0
			s.bitbuf2 = 0
		}
	}
}

func (s *SGB) receivePacket() {
	var pkt [16]byte
	for i := 0; i < 8; i++ {
		pkt[i] = byte(s.bitbuf >> (i * 8))
	}
	for i := 0; i < 8; i++ {
		pkt[8+i] = byte(s.bitbuf2 >> (i * 8))
	}

	if s.pktcnt == 0 {
		// First packet: decode command and length
		cmd := pkt[0] >> 3
		length := pkt[0] & 0x07
		if length == 0 {
			length = 1
		}
		s.pktlen = int(length)
		_ = cmd
	}
	s.packets[s.pktcnt] = pkt
	s.pktcnt++

	if s.pktcnt >= s.pktlen {
		s.processCommand()
		s.pktcnt = 0
		s.pktlen = 0
	}
}

func (s *SGB) processCommand() {
	cmd := s.packets[0][0] >> 3
	data := s.packets[0][1:] // first packet payload

	switch cmd {
	case CmdPAL01:
		s.setPalettePair(0, 1, data)
	case CmdPAL23:
		s.setPalettePair(2, 3, data)
	case CmdPAL03:
		s.setPalettePair(0, 3, data)
	case CmdPAL12:
		s.setPalettePair(1, 2, data)

	case CmdATF_SET:
		s.handleATF_SET(data)

	case CmdPAL_SET:
		s.handlePAL_SET(data)

	case CmdMASK_EN:
		s.MaskMode = data[0] & 0x03

	// CHR_TRN / PCT_TRN / PAL_TRN are handled via VRAM snapshot
	// (caller must call TransferVRAM after these commands)
	case CmdCHR_TRN, CmdPCT_TRN, CmdPAL_TRN:
		// Caller triggers VRAM transfer
	}
}

// setPalettePair decodes two 4-color palettes from 8 bytes of BGR555 data
func (s *SGB) setPalettePair(p1, p2 int, data []byte) {
	for i := 0; i < 4; i++ {
		s.Palettes[p1][i] = bgr555ToRGBA(data[i*2], data[i*2+1])
	}
	for i := 0; i < 4; i++ {
		s.Palettes[p2][i] = bgr555ToRGBA(data[8+i*2], data[8+i*2+1])
	}
}

func (s *SGB) handleATF_SET(data []byte) {
	// 90 bytes of ATF data packed as 2 bits per tile
	idx := 0
	for byte_i := 0; byte_i < 90 && idx < 360; byte_i++ {
		b := data[byte_i]
		for bit := 0; bit < 4 && idx < 360; bit++ {
			s.ATF[idx] = (b >> (bit * 2)) & 0x03
			idx++
		}
	}
}

func (s *SGB) handlePAL_SET(data []byte) {
	// Select 4 palettes from SysPalRAM and optionally apply ATF
	for i := 0; i < 4; i++ {
		palIdx := int(data[i*2]) | int(data[i*2+1]&0x01)<<8
		if palIdx*8+8 <= len(s.SysPalRAM) {
			for c := 0; c < 4; c++ {
				lo := s.SysPalRAM[palIdx*8+c*2]
				hi := s.SysPalRAM[palIdx*8+c*2+1]
				s.Palettes[i][c] = bgr555ToRGBA(lo, hi)
			}
		}
	}
	// bit6 of data[8] = apply ATF
	if data[8]&0x40 != 0 {
		// ATF index in data[8] bits 0-5
	}
}

// TransferVRAM is called after CHR_TRN / PCT_TRN / PAL_TRN commands.
// vram is the full 8KB VRAM snapshot. upper=true for second 4KB bank.
func (s *SGB) TransferVRAM(cmd byte, vram []byte, upper bool) {
	switch cmd {
	case CmdCHR_TRN:
		offset := 0
		if upper {
			offset = 0x1000
		}
		copy(s.BorderTiles[offset:], vram[:0x1000])
	case CmdPCT_TRN:
		copy(s.BorderMap[:], vram[:0x1000])
		s.BorderReady = true
	case CmdPAL_TRN:
		copy(s.SysPalRAM[:], vram[:len(s.SysPalRAM)])
	}
}

// PaletteForTile returns the SGB palette index for the given BG tile position
func (s *SGB) PaletteForTile(tileX, tileY int) int {
	if tileX >= 20 || tileY >= 18 {
		return 0
	}
	return int(s.ATF[tileY*20+tileX])
}

// ColorForIndex converts a 2-bit DMG color index + tile position to RGBA
func (s *SGB) ColorForIndex(colorIdx byte, tileX, tileY int) color.RGBA {
	palIdx := s.PaletteForTile(tileX, tileY)
	return s.Palettes[palIdx][colorIdx&3]
}

// bgr555ToRGBA converts a 15-bit BGR555 little-endian word to color.RGBA
func bgr555ToRGBA(lo, hi byte) color.RGBA {
	word := uint16(lo) | uint16(hi)<<8
	r := byte((word & 0x001F) << 3)
	g := byte(((word >> 5) & 0x1F) << 3)
	b := byte(((word >> 10) & 0x1F) << 3)
	return color.RGBA{r | r >> 5, g | g >> 5, b | b >> 5, 0xFF}
}
