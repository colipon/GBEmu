package cartridge

import (
	"fmt"
	"os"
)

// MBC type constants
const (
	MBCNone = iota
	MBC1
	MBC2
	MBC3
	MBC5
)

// Cartridge represents a Game Boy ROM cartridge
type Cartridge struct {
	ROM  []byte
	RAM  []byte
	mbc  int
	romBank int
	ramBank int
	ramEnabled bool
	mode int // MBC1 mode
}

// Load reads a ROM file and returns a Cartridge
func Load(path string) (*Cartridge, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read ROM: %w", err)
	}
	if len(data) < 0x150 {
		return nil, fmt.Errorf("ROM too small")
	}

	c := &Cartridge{
		ROM:     data,
		romBank: 1,
	}

	mbcType := data[0x147]
	switch mbcType {
	case 0x00:
		c.mbc = MBCNone
	case 0x01, 0x02, 0x03:
		c.mbc = MBC1
	case 0x05, 0x06:
		c.mbc = MBC2
	case 0x0F, 0x10, 0x11, 0x12, 0x13:
		c.mbc = MBC3
	case 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E:
		c.mbc = MBC5
	default:
		c.mbc = MBCNone
	}

	ramSizes := []int{0, 0, 8 * 1024, 32 * 1024, 128 * 1024, 64 * 1024}
	ramIdx := int(data[0x149])
	if ramIdx < len(ramSizes) {
		c.RAM = make([]byte, ramSizes[ramIdx])
	}
	if c.mbc == MBC2 {
		c.RAM = make([]byte, 512)
	}
	if len(c.RAM) == 0 {
		c.RAM = make([]byte, 8*1024) // minimum
	}

	title := string(data[0x134:0x144])
	fmt.Printf("[Cart] Loaded: %q  MBC=%d  ROM=%dKB  RAM=%dB\n",
		title, c.mbc, len(data)/1024, len(c.RAM))
	return c, nil
}

// Read handles cartridge reads
func (c *Cartridge) Read(addr uint16) byte {
	switch {
	case addr < 0x4000:
		return c.ROM[addr]
	case addr < 0x8000:
		offset := int(c.romBank) * 0x4000
		idx := offset + int(addr-0x4000)
		if idx < len(c.ROM) {
			return c.ROM[idx]
		}
		return 0xFF
	case addr >= 0xA000 && addr < 0xC000:
		if !c.ramEnabled || len(c.RAM) == 0 {
			return 0xFF
		}
		offset := int(c.ramBank) * 0x2000
		idx := offset + int(addr-0xA000)
		if idx < len(c.RAM) {
			return c.RAM[idx]
		}
		return 0xFF
	}
	return 0xFF
}

// Write handles MBC register writes
func (c *Cartridge) Write(addr uint16, val byte) {
	switch c.mbc {
	case MBCNone:
		// no-op
	case MBC1:
		c.writeMBC1(addr, val)
	case MBC3:
		c.writeMBC3(addr, val)
	case MBC5:
		c.writeMBC5(addr, val)
	}
}

func (c *Cartridge) writeMBC1(addr uint16, val byte) {
	switch {
	case addr < 0x2000:
		c.ramEnabled = (val & 0x0F) == 0x0A
	case addr < 0x4000:
		bank := int(val & 0x1F)
		if bank == 0 {
			bank = 1
		}
		c.romBank = (c.romBank & 0x60) | bank
	case addr < 0x6000:
		if c.mode == 0 {
			c.romBank = (c.romBank & 0x1F) | (int(val&3) << 5)
		} else {
			c.ramBank = int(val & 3)
		}
	case addr < 0x8000:
		c.mode = int(val & 1)
	case addr >= 0xA000 && addr < 0xC000:
		if c.ramEnabled {
			offset := int(c.ramBank) * 0x2000
			idx := offset + int(addr-0xA000)
			if idx < len(c.RAM) {
				c.RAM[idx] = val
			}
		}
	}
}

func (c *Cartridge) writeMBC3(addr uint16, val byte) {
	switch {
	case addr < 0x2000:
		c.ramEnabled = (val & 0x0F) == 0x0A
	case addr < 0x4000:
		bank := int(val & 0x7F)
		if bank == 0 {
			bank = 1
		}
		c.romBank = bank
	case addr < 0x6000:
		if val <= 3 {
			c.ramBank = int(val)
		}
	case addr >= 0xA000 && addr < 0xC000:
		if c.ramEnabled {
			idx := int(c.ramBank)*0x2000 + int(addr-0xA000)
			if idx < len(c.RAM) {
				c.RAM[idx] = val
			}
		}
	}
}

func (c *Cartridge) writeMBC5(addr uint16, val byte) {
	switch {
	case addr < 0x2000:
		c.ramEnabled = (val & 0x0F) == 0x0A
	case addr < 0x3000:
		c.romBank = (c.romBank & 0x100) | int(val)
	case addr < 0x4000:
		c.romBank = (c.romBank & 0xFF) | (int(val&1) << 8)
	case addr < 0x6000:
		c.ramBank = int(val & 0x0F)
	case addr >= 0xA000 && addr < 0xC000:
		if c.ramEnabled {
			idx := int(c.ramBank)*0x2000 + int(addr-0xA000)
			if idx < len(c.RAM) {
				c.RAM[idx] = val
			}
		}
	}
}

// Title returns the ROM title string
func (c *Cartridge) Title() string {
	raw := c.ROM[0x134:0x144]
	end := 0
	for end < len(raw) && raw[end] != 0 {
		end++
	}
	return string(raw[:end])
}
