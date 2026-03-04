package cpu

import "github.com/user/gbemu/internal/mmu"

// Flag bit positions
const (
	FlagZ = 7
	FlagN = 6
	FlagH = 5
	FlagC = 4
)

// CPU represents the Sharp LR35902 processor
type CPU struct {
	// Registers
	A, F byte
	B, C byte
	D, E byte
	H, L byte
	SP   uint16
	PC   uint16

	// State
	IME     bool // Interrupt Master Enable
	Halted  bool
	Stopped bool
	cycles  int // cycles spent this step

	mem *mmu.MMU
}

// New creates a CPU in post-boot-ROM state (DMG)
func New(mem *mmu.MMU) *CPU {
	c := &CPU{mem: mem}
	c.A = 0x01; c.F = 0xB0
	c.B = 0x00; c.C = 0x13
	c.D = 0x00; c.E = 0xD8
	c.H = 0x01; c.L = 0x4D
	c.SP = 0xFFFE
	c.PC = 0x0100
	return c
}

// ── Register pairs ────────────────────────────────────────────────────────────

func (c *CPU) AF() uint16 { return uint16(c.A)<<8 | uint16(c.F) }
func (c *CPU) BC() uint16 { return uint16(c.B)<<8 | uint16(c.C) }
func (c *CPU) DE() uint16 { return uint16(c.D)<<8 | uint16(c.E) }
func (c *CPU) HL() uint16 { return uint16(c.H)<<8 | uint16(c.L) }

func (c *CPU) SetAF(v uint16) { c.A = byte(v >> 8); c.F = byte(v) & 0xF0 }
func (c *CPU) SetBC(v uint16) { c.B = byte(v >> 8); c.C = byte(v) }
func (c *CPU) SetDE(v uint16) { c.D = byte(v >> 8); c.E = byte(v) }
func (c *CPU) SetHL(v uint16) { c.H = byte(v >> 8); c.L = byte(v) }

// ── Flags ─────────────────────────────────────────────────────────────────────

func (c *CPU) getFlag(bit int) bool   { return c.F>>bit&1 == 1 }
func (c *CPU) setFlag(bit int, v bool) {
	if v {
		c.F |= 1 << bit
	} else {
		c.F &^= 1 << bit
	}
}

func (c *CPU) flagZ() bool { return c.getFlag(FlagZ) }
func (c *CPU) flagN() bool { return c.getFlag(FlagN) }
func (c *CPU) flagH() bool { return c.getFlag(FlagH) }
func (c *CPU) flagC() bool { return c.getFlag(FlagC) }

// ── Memory helpers ────────────────────────────────────────────────────────────

func (c *CPU) fetch() byte {
	v := c.mem.Read(c.PC)
	c.PC++
	return v
}

func (c *CPU) fetch16() uint16 {
	lo := uint16(c.fetch())
	hi := uint16(c.fetch())
	return hi<<8 | lo
}

func (c *CPU) push(val uint16) {
	c.SP -= 2
	c.mem.Write16(c.SP, val)
}

func (c *CPU) pop() uint16 {
	v := c.mem.Read16(c.SP)
	c.SP += 2
	return v
}

// ── ALU helpers ───────────────────────────────────────────────────────────────

func (c *CPU) add8(a, b byte) byte {
	res := uint16(a) + uint16(b)
	c.setFlag(FlagZ, byte(res) == 0)
	c.setFlag(FlagN, false)
	c.setFlag(FlagH, (a&0xF)+(b&0xF) > 0xF)
	c.setFlag(FlagC, res > 0xFF)
	return byte(res)
}

func (c *CPU) adc8(a, b byte) byte {
	carry := byte(0)
	if c.flagC() { carry = 1 }
	res := uint16(a) + uint16(b) + uint16(carry)
	c.setFlag(FlagZ, byte(res) == 0)
	c.setFlag(FlagN, false)
	c.setFlag(FlagH, (a&0xF)+(b&0xF)+carry > 0xF)
	c.setFlag(FlagC, res > 0xFF)
	return byte(res)
}

func (c *CPU) sub8(a, b byte) byte {
	res := int(a) - int(b)
	c.setFlag(FlagZ, byte(res) == 0)
	c.setFlag(FlagN, true)
	c.setFlag(FlagH, int(a&0xF)-int(b&0xF) < 0)
	c.setFlag(FlagC, res < 0)
	return byte(res)
}

func (c *CPU) sbc8(a, b byte) byte {
	carry := byte(0)
	if c.flagC() { carry = 1 }
	res := int(a) - int(b) - int(carry)
	c.setFlag(FlagZ, byte(res) == 0)
	c.setFlag(FlagN, true)
	c.setFlag(FlagH, int(a&0xF)-int(b&0xF)-int(carry) < 0)
	c.setFlag(FlagC, res < 0)
	return byte(res)
}

func (c *CPU) and8(a, b byte) byte {
	res := a & b
	c.setFlag(FlagZ, res == 0)
	c.setFlag(FlagN, false)
	c.setFlag(FlagH, true)
	c.setFlag(FlagC, false)
	return res
}

func (c *CPU) or8(a, b byte) byte {
	res := a | b
	c.setFlag(FlagZ, res == 0)
	c.setFlag(FlagN, false)
	c.setFlag(FlagH, false)
	c.setFlag(FlagC, false)
	return res
}

func (c *CPU) xor8(a, b byte) byte {
	res := a ^ b
	c.setFlag(FlagZ, res == 0)
	c.setFlag(FlagN, false)
	c.setFlag(FlagH, false)
	c.setFlag(FlagC, false)
	return res
}

func (c *CPU) cp8(a, b byte) {
	c.sub8(a, b)
}

func (c *CPU) inc8(a byte) byte {
	res := a + 1
	c.setFlag(FlagZ, res == 0)
	c.setFlag(FlagN, false)
	c.setFlag(FlagH, (a&0xF) == 0xF)
	return res
}

func (c *CPU) dec8(a byte) byte {
	res := a - 1
	c.setFlag(FlagZ, res == 0)
	c.setFlag(FlagN, true)
	c.setFlag(FlagH, (a&0xF) == 0x00)
	return res
}

func (c *CPU) addHL(v uint16) {
	hl := c.HL()
	res := uint32(hl) + uint32(v)
	c.setFlag(FlagN, false)
	c.setFlag(FlagH, (hl&0xFFF)+(v&0xFFF) > 0xFFF)
	c.setFlag(FlagC, res > 0xFFFF)
	c.SetHL(uint16(res))
}

// ── Step ──────────────────────────────────────────────────────────────────────

// Step executes one instruction and returns the number of T-cycles consumed
func (c *CPU) Step() int {
	// Handle interrupts
	if c.Halted {
		if c.mem.IF&c.mem.IE&0x1F != 0 {
			c.Halted = false
		} else {
			return 4
		}
	}

	if c.IME {
		if fired := c.handleInterrupts(); fired {
			return 20
		}
	}

	c.cycles = 0
	op := c.fetch()
	c.execute(op)
	return c.cycles
}

func (c *CPU) handleInterrupts() bool {
	pending := c.mem.IF & c.mem.IE & 0x1F
	if pending == 0 {
		return false
	}
	c.IME = false
	for bit := 0; bit < 5; bit++ {
		if pending>>bit&1 == 1 {
			c.mem.IF &^= 1 << bit
			vectors := [5]uint16{0x40, 0x48, 0x50, 0x58, 0x60}
			c.push(c.PC)
			c.PC = vectors[bit]
			return true
		}
	}
	return false
}
