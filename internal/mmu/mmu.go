package mmu

import (
	"github.com/colipon/gbemu/internal/cartridge"
)

// Interrupt flag bits
const (
	IntVBlank  = 0x01
	IntLCDStat = 0x02
	IntTimer   = 0x04
	IntSerial  = 0x08
	IntJoypad  = 0x10
)

// MMU holds all addressable memory for the Game Boy
type MMU struct {
	Cart    *cartridge.Cartridge
	VRAM    [0x2000]byte // 0x8000-0x9FFF
	WRAM    [0x2000]byte // 0xC000-0xDFFF
	OAM     [0xA0]byte  // 0xFE00-0xFE9F
	HRAM    [0x7F]byte  // 0xFF80-0xFFFE
	IO      [0x80]byte  // 0xFF00-0xFF7F
	IE      byte        // 0xFFFF
	IF      byte        // 0xFF0F
	bootROM [256]byte
	bootEnabled bool

	// Callbacks for I/O side-effects
	OnTimerWrite  func(addr uint16, val byte)
	OnPPUWrite    func(addr uint16, val byte)
	OnJoypadRead  func() byte
	OnJoypadWrite func(val byte) // SGB packet detection
	OnDMALaunch   func(src byte)
}

// New creates an MMU with post-boot state
func New(cart *cartridge.Cartridge) *MMU {
	m := &MMU{Cart: cart}
	m.initPostBoot()
	return m
}

func (m *MMU) initPostBoot() {
	// Simulate DMG boot ROM finishing
	m.IO[0x05] = 0x00 // TIMA
	m.IO[0x06] = 0x00 // TMA
	m.IO[0x07] = 0x00 // TAC
	m.IO[0x10] = 0x80
	m.IO[0x11] = 0xBF
	m.IO[0x12] = 0xF3
	m.IO[0x14] = 0xBF
	m.IO[0x16] = 0x3F
	m.IO[0x19] = 0xBF
	m.IO[0x1A] = 0x7F
	m.IO[0x1B] = 0xFF
	m.IO[0x1C] = 0x9F
	m.IO[0x1E] = 0xBF
	m.IO[0x20] = 0xFF
	m.IO[0x23] = 0xBF
	m.IO[0x24] = 0x77
	m.IO[0x25] = 0xF3
	m.IO[0x26] = 0xF1
	m.IO[0x40] = 0x91 // LCDC
	m.IO[0x42] = 0x00 // SCY
	m.IO[0x43] = 0x00 // SCX
	m.IO[0x45] = 0x00 // LYC
	m.IO[0x47] = 0xFC // BGP
	m.IO[0x48] = 0xFF // OBP0
	m.IO[0x49] = 0xFF // OBP1
	m.IO[0x4A] = 0x00 // WY
	m.IO[0x4B] = 0x00 // WX
	m.IF = 0xE1
}

// Read returns the byte at the given address
func (m *MMU) Read(addr uint16) byte {
	switch {
	case addr < 0x8000:
		return m.Cart.Read(addr)
	case addr < 0xA000:
		return m.VRAM[addr-0x8000]
	case addr < 0xC000:
		return m.Cart.Read(addr)
	case addr < 0xE000:
		return m.WRAM[addr-0xC000]
	case addr < 0xFE00:
		return m.WRAM[addr-0xE000] // echo
	case addr < 0xFEA0:
		return m.OAM[addr-0xFE00]
	case addr < 0xFF00:
		return 0xFF // unusable
	case addr < 0xFF80:
		return m.readIO(addr)
	case addr < 0xFFFF:
		return m.HRAM[addr-0xFF80]
	case addr == 0xFFFF:
		return m.IE
	}
	return 0xFF
}

// Write stores a byte at the given address
func (m *MMU) Write(addr uint16, val byte) {
	switch {
	case addr < 0x8000:
		m.Cart.Write(addr, val)
	case addr < 0xA000:
		m.VRAM[addr-0x8000] = val
		if m.OnPPUWrite != nil {
			m.OnPPUWrite(addr, val)
		}
	case addr < 0xC000:
		m.Cart.Write(addr, val)
	case addr < 0xE000:
		m.WRAM[addr-0xC000] = val
	case addr < 0xFE00:
		m.WRAM[addr-0xE000] = val
	case addr < 0xFEA0:
		m.OAM[addr-0xFE00] = val
	case addr < 0xFF00:
		// unusable
	case addr < 0xFF80:
		m.writeIO(addr, val)
	case addr < 0xFFFF:
		m.HRAM[addr-0xFF80] = val
	case addr == 0xFFFF:
		m.IE = val
	}
}

func (m *MMU) readIO(addr uint16) byte {
	reg := addr - 0xFF00
	switch addr {
	case 0xFF00: // Joypad
		if m.OnJoypadRead != nil {
			return m.OnJoypadRead()
		}
		return 0xFF
	case 0xFF0F:
		return m.IF
	}
	return m.IO[reg]
}

func (m *MMU) writeIO(addr uint16, val byte) {
	reg := addr - 0xFF00
	switch addr {
	case 0xFF00: // Joypad select write
		m.IO[reg] = val
		if m.OnJoypadWrite != nil {
			m.OnJoypadWrite(val)
		}
		return
	case 0xFF46: // DMA
		m.IO[reg] = val
		if m.OnDMALaunch != nil {
			m.OnDMALaunch(val)
		}
		return
	}
	m.IO[reg] = val

	if addr >= 0xFF04 && addr <= 0xFF07 && m.OnTimerWrite != nil {
		m.OnTimerWrite(addr, val)
	}
	if (addr >= 0xFF40 && addr <= 0xFF4B) && m.OnPPUWrite != nil {
		m.OnPPUWrite(addr, val)
	}
}

// Read16 reads a little-endian 16-bit value
func (m *MMU) Read16(addr uint16) uint16 {
	lo := uint16(m.Read(addr))
	hi := uint16(m.Read(addr + 1))
	return (hi << 8) | lo
}

// Write16 writes a little-endian 16-bit value
func (m *MMU) Write16(addr uint16, val uint16) {
	m.Write(addr, byte(val))
	m.Write(addr+1, byte(val>>8))
}

// RequestInterrupt sets an interrupt flag
func (m *MMU) RequestInterrupt(bit byte) {
	m.IF |= bit
}

// DMA performs an OAM DMA transfer
func (m *MMU) DMA(src byte) {
	base := uint16(src) << 8
	for i := uint16(0); i < 0xA0; i++ {
		m.OAM[i] = m.Read(base + i)
	}
}
