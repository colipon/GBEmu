package cpu

// execute dispatches the main opcode table
func (c *CPU) execute(op byte) {
	switch op {
	// ── NOP / misc ───────────────────────────────────────────────────────────
	case 0x00: c.cycles += 4 // NOP
	case 0x76: c.Halted = true; c.cycles += 4 // HALT
	case 0xF3: c.IME = false; c.cycles += 4   // DI
	case 0xFB: c.IME = true; c.cycles += 4    // EI

	// ── 8-bit loads ─────────────────────────────────────────────────────────
	case 0x06: c.B = c.fetch(); c.cycles += 8
	case 0x0E: c.C = c.fetch(); c.cycles += 8
	case 0x16: c.D = c.fetch(); c.cycles += 8
	case 0x1E: c.E = c.fetch(); c.cycles += 8
	case 0x26: c.H = c.fetch(); c.cycles += 8
	case 0x2E: c.L = c.fetch(); c.cycles += 8
	case 0x3E: c.A = c.fetch(); c.cycles += 8
	case 0x36: c.mem.Write(c.HL(), c.fetch()); c.cycles += 12

	// LD r,r (rows 0x40-0x7F, minus 0x76 HALT)
	case 0x40: c.cycles += 4 // LD B,B
	case 0x41: c.B = c.C; c.cycles += 4
	case 0x42: c.B = c.D; c.cycles += 4
	case 0x43: c.B = c.E; c.cycles += 4
	case 0x44: c.B = c.H; c.cycles += 4
	case 0x45: c.B = c.L; c.cycles += 4
	case 0x46: c.B = c.mem.Read(c.HL()); c.cycles += 8
	case 0x47: c.B = c.A; c.cycles += 4
	case 0x48: c.C = c.B; c.cycles += 4
	case 0x49: c.cycles += 4
	case 0x4A: c.C = c.D; c.cycles += 4
	case 0x4B: c.C = c.E; c.cycles += 4
	case 0x4C: c.C = c.H; c.cycles += 4
	case 0x4D: c.C = c.L; c.cycles += 4
	case 0x4E: c.C = c.mem.Read(c.HL()); c.cycles += 8
	case 0x4F: c.C = c.A; c.cycles += 4
	case 0x50: c.D = c.B; c.cycles += 4
	case 0x51: c.D = c.C; c.cycles += 4
	case 0x52: c.cycles += 4
	case 0x53: c.D = c.E; c.cycles += 4
	case 0x54: c.D = c.H; c.cycles += 4
	case 0x55: c.D = c.L; c.cycles += 4
	case 0x56: c.D = c.mem.Read(c.HL()); c.cycles += 8
	case 0x57: c.D = c.A; c.cycles += 4
	case 0x58: c.E = c.B; c.cycles += 4
	case 0x59: c.E = c.C; c.cycles += 4
	case 0x5A: c.E = c.D; c.cycles += 4
	case 0x5B: c.cycles += 4
	case 0x5C: c.E = c.H; c.cycles += 4
	case 0x5D: c.E = c.L; c.cycles += 4
	case 0x5E: c.E = c.mem.Read(c.HL()); c.cycles += 8
	case 0x5F: c.E = c.A; c.cycles += 4
	case 0x60: c.H = c.B; c.cycles += 4
	case 0x61: c.H = c.C; c.cycles += 4
	case 0x62: c.H = c.D; c.cycles += 4
	case 0x63: c.H = c.E; c.cycles += 4
	case 0x64: c.cycles += 4
	case 0x65: c.H = c.L; c.cycles += 4
	case 0x66: c.H = c.mem.Read(c.HL()); c.cycles += 8
	case 0x67: c.H = c.A; c.cycles += 4
	case 0x68: c.L = c.B; c.cycles += 4
	case 0x69: c.L = c.C; c.cycles += 4
	case 0x6A: c.L = c.D; c.cycles += 4
	case 0x6B: c.L = c.E; c.cycles += 4
	case 0x6C: c.L = c.H; c.cycles += 4
	case 0x6D: c.cycles += 4
	case 0x6E: c.L = c.mem.Read(c.HL()); c.cycles += 8
	case 0x6F: c.L = c.A; c.cycles += 4
	case 0x70: c.mem.Write(c.HL(), c.B); c.cycles += 8
	case 0x71: c.mem.Write(c.HL(), c.C); c.cycles += 8
	case 0x72: c.mem.Write(c.HL(), c.D); c.cycles += 8
	case 0x73: c.mem.Write(c.HL(), c.E); c.cycles += 8
	case 0x74: c.mem.Write(c.HL(), c.H); c.cycles += 8
	case 0x75: c.mem.Write(c.HL(), c.L); c.cycles += 8
	case 0x77: c.mem.Write(c.HL(), c.A); c.cycles += 8
	case 0x78: c.A = c.B; c.cycles += 4
	case 0x79: c.A = c.C; c.cycles += 4
	case 0x7A: c.A = c.D; c.cycles += 4
	case 0x7B: c.A = c.E; c.cycles += 4
	case 0x7C: c.A = c.H; c.cycles += 4
	case 0x7D: c.A = c.L; c.cycles += 4
	case 0x7E: c.A = c.mem.Read(c.HL()); c.cycles += 8
	case 0x7F: c.cycles += 4

	// Special LD
	case 0x02: c.mem.Write(c.BC(), c.A); c.cycles += 8
	case 0x12: c.mem.Write(c.DE(), c.A); c.cycles += 8
	case 0x22: c.mem.Write(c.HL(), c.A); c.SetHL(c.HL()+1); c.cycles += 8
	case 0x32: c.mem.Write(c.HL(), c.A); c.SetHL(c.HL()-1); c.cycles += 8
	case 0x0A: c.A = c.mem.Read(c.BC()); c.cycles += 8
	case 0x1A: c.A = c.mem.Read(c.DE()); c.cycles += 8
	case 0x2A: c.A = c.mem.Read(c.HL()); c.SetHL(c.HL()+1); c.cycles += 8
	case 0x3A: c.A = c.mem.Read(c.HL()); c.SetHL(c.HL()-1); c.cycles += 8

	// IO loads
	case 0xE0: c.mem.Write(0xFF00|uint16(c.fetch()), c.A); c.cycles += 12
	case 0xF0: c.A = c.mem.Read(0xFF00 | uint16(c.fetch())); c.cycles += 12
	case 0xE2: c.mem.Write(0xFF00|uint16(c.C), c.A); c.cycles += 8
	case 0xF2: c.A = c.mem.Read(0xFF00 | uint16(c.C)); c.cycles += 8
	case 0xEA: c.mem.Write(c.fetch16(), c.A); c.cycles += 16
	case 0xFA: c.A = c.mem.Read(c.fetch16()); c.cycles += 16

	// ── 16-bit loads ─────────────────────────────────────────────────────────
	case 0x01: c.SetBC(c.fetch16()); c.cycles += 12
	case 0x11: c.SetDE(c.fetch16()); c.cycles += 12
	case 0x21: c.SetHL(c.fetch16()); c.cycles += 12
	case 0x31: c.SP = c.fetch16(); c.cycles += 12
	case 0x08:
		addr := c.fetch16()
		c.mem.Write16(addr, c.SP)
		c.cycles += 20
	case 0xF9: c.SP = c.HL(); c.cycles += 8
	case 0xF8:
		e := int8(c.fetch())
		res := int32(c.SP) + int32(e)
		c.setFlag(FlagZ, false)
		c.setFlag(FlagN, false)
		c.setFlag(FlagH, int(c.SP&0xF)+int(byte(e)&0xF) > 0xF)
		c.setFlag(FlagC, int(c.SP&0xFF)+int(byte(e)) > 0xFF)
		c.SetHL(uint16(res))
		c.cycles += 12
	case 0xC5: c.push(c.BC()); c.cycles += 16
	case 0xD5: c.push(c.DE()); c.cycles += 16
	case 0xE5: c.push(c.HL()); c.cycles += 16
	case 0xF5: c.push(c.AF()); c.cycles += 16
	case 0xC1: c.SetBC(c.pop()); c.cycles += 12
	case 0xD1: c.SetDE(c.pop()); c.cycles += 12
	case 0xE1: c.SetHL(c.pop()); c.cycles += 12
	case 0xF1: c.SetAF(c.pop()); c.cycles += 12

	// ── 8-bit ALU ────────────────────────────────────────────────────────────
	case 0x80: c.A = c.add8(c.A, c.B); c.cycles += 4
	case 0x81: c.A = c.add8(c.A, c.C); c.cycles += 4
	case 0x82: c.A = c.add8(c.A, c.D); c.cycles += 4
	case 0x83: c.A = c.add8(c.A, c.E); c.cycles += 4
	case 0x84: c.A = c.add8(c.A, c.H); c.cycles += 4
	case 0x85: c.A = c.add8(c.A, c.L); c.cycles += 4
	case 0x86: c.A = c.add8(c.A, c.mem.Read(c.HL())); c.cycles += 8
	case 0x87: c.A = c.add8(c.A, c.A); c.cycles += 4
	case 0x88: c.A = c.adc8(c.A, c.B); c.cycles += 4
	case 0x89: c.A = c.adc8(c.A, c.C); c.cycles += 4
	case 0x8A: c.A = c.adc8(c.A, c.D); c.cycles += 4
	case 0x8B: c.A = c.adc8(c.A, c.E); c.cycles += 4
	case 0x8C: c.A = c.adc8(c.A, c.H); c.cycles += 4
	case 0x8D: c.A = c.adc8(c.A, c.L); c.cycles += 4
	case 0x8E: c.A = c.adc8(c.A, c.mem.Read(c.HL())); c.cycles += 8
	case 0x8F: c.A = c.adc8(c.A, c.A); c.cycles += 4
	case 0x90: c.A = c.sub8(c.A, c.B); c.cycles += 4
	case 0x91: c.A = c.sub8(c.A, c.C); c.cycles += 4
	case 0x92: c.A = c.sub8(c.A, c.D); c.cycles += 4
	case 0x93: c.A = c.sub8(c.A, c.E); c.cycles += 4
	case 0x94: c.A = c.sub8(c.A, c.H); c.cycles += 4
	case 0x95: c.A = c.sub8(c.A, c.L); c.cycles += 4
	case 0x96: c.A = c.sub8(c.A, c.mem.Read(c.HL())); c.cycles += 8
	case 0x97: c.A = c.sub8(c.A, c.A); c.cycles += 4
	case 0x98: c.A = c.sbc8(c.A, c.B); c.cycles += 4
	case 0x99: c.A = c.sbc8(c.A, c.C); c.cycles += 4
	case 0x9A: c.A = c.sbc8(c.A, c.D); c.cycles += 4
	case 0x9B: c.A = c.sbc8(c.A, c.E); c.cycles += 4
	case 0x9C: c.A = c.sbc8(c.A, c.H); c.cycles += 4
	case 0x9D: c.A = c.sbc8(c.A, c.L); c.cycles += 4
	case 0x9E: c.A = c.sbc8(c.A, c.mem.Read(c.HL())); c.cycles += 8
	case 0x9F: c.A = c.sbc8(c.A, c.A); c.cycles += 4
	case 0xA0: c.A = c.and8(c.A, c.B); c.cycles += 4
	case 0xA1: c.A = c.and8(c.A, c.C); c.cycles += 4
	case 0xA2: c.A = c.and8(c.A, c.D); c.cycles += 4
	case 0xA3: c.A = c.and8(c.A, c.E); c.cycles += 4
	case 0xA4: c.A = c.and8(c.A, c.H); c.cycles += 4
	case 0xA5: c.A = c.and8(c.A, c.L); c.cycles += 4
	case 0xA6: c.A = c.and8(c.A, c.mem.Read(c.HL())); c.cycles += 8
	case 0xA7: c.A = c.and8(c.A, c.A); c.cycles += 4
	case 0xA8: c.A = c.xor8(c.A, c.B); c.cycles += 4
	case 0xA9: c.A = c.xor8(c.A, c.C); c.cycles += 4
	case 0xAA: c.A = c.xor8(c.A, c.D); c.cycles += 4
	case 0xAB: c.A = c.xor8(c.A, c.E); c.cycles += 4
	case 0xAC: c.A = c.xor8(c.A, c.H); c.cycles += 4
	case 0xAD: c.A = c.xor8(c.A, c.L); c.cycles += 4
	case 0xAE: c.A = c.xor8(c.A, c.mem.Read(c.HL())); c.cycles += 8
	case 0xAF: c.A = c.xor8(c.A, c.A); c.cycles += 4
	case 0xB0: c.A = c.or8(c.A, c.B); c.cycles += 4
	case 0xB1: c.A = c.or8(c.A, c.C); c.cycles += 4
	case 0xB2: c.A = c.or8(c.A, c.D); c.cycles += 4
	case 0xB3: c.A = c.or8(c.A, c.E); c.cycles += 4
	case 0xB4: c.A = c.or8(c.A, c.H); c.cycles += 4
	case 0xB5: c.A = c.or8(c.A, c.L); c.cycles += 4
	case 0xB6: c.A = c.or8(c.A, c.mem.Read(c.HL())); c.cycles += 8
	case 0xB7: c.A = c.or8(c.A, c.A); c.cycles += 4
	case 0xB8: c.cp8(c.A, c.B); c.cycles += 4
	case 0xB9: c.cp8(c.A, c.C); c.cycles += 4
	case 0xBA: c.cp8(c.A, c.D); c.cycles += 4
	case 0xBB: c.cp8(c.A, c.E); c.cycles += 4
	case 0xBC: c.cp8(c.A, c.H); c.cycles += 4
	case 0xBD: c.cp8(c.A, c.L); c.cycles += 4
	case 0xBE: c.cp8(c.A, c.mem.Read(c.HL())); c.cycles += 8
	case 0xBF: c.cp8(c.A, c.A); c.cycles += 4
	case 0xC6: c.A = c.add8(c.A, c.fetch()); c.cycles += 8
	case 0xCE: c.A = c.adc8(c.A, c.fetch()); c.cycles += 8
	case 0xD6: c.A = c.sub8(c.A, c.fetch()); c.cycles += 8
	case 0xDE: c.A = c.sbc8(c.A, c.fetch()); c.cycles += 8
	case 0xE6: c.A = c.and8(c.A, c.fetch()); c.cycles += 8
	case 0xEE: c.A = c.xor8(c.A, c.fetch()); c.cycles += 8
	case 0xF6: c.A = c.or8(c.A, c.fetch()); c.cycles += 8
	case 0xFE: c.cp8(c.A, c.fetch()); c.cycles += 8

	// ── INC / DEC 8-bit ──────────────────────────────────────────────────────
	case 0x04: c.B = c.inc8(c.B); c.cycles += 4
	case 0x0C: c.C = c.inc8(c.C); c.cycles += 4
	case 0x14: c.D = c.inc8(c.D); c.cycles += 4
	case 0x1C: c.E = c.inc8(c.E); c.cycles += 4
	case 0x24: c.H = c.inc8(c.H); c.cycles += 4
	case 0x2C: c.L = c.inc8(c.L); c.cycles += 4
	case 0x34: c.mem.Write(c.HL(), c.inc8(c.mem.Read(c.HL()))); c.cycles += 12
	case 0x3C: c.A = c.inc8(c.A); c.cycles += 4
	case 0x05: c.B = c.dec8(c.B); c.cycles += 4
	case 0x0D: c.C = c.dec8(c.C); c.cycles += 4
	case 0x15: c.D = c.dec8(c.D); c.cycles += 4
	case 0x1D: c.E = c.dec8(c.E); c.cycles += 4
	case 0x25: c.H = c.dec8(c.H); c.cycles += 4
	case 0x2D: c.L = c.dec8(c.L); c.cycles += 4
	case 0x35: c.mem.Write(c.HL(), c.dec8(c.mem.Read(c.HL()))); c.cycles += 12
	case 0x3D: c.A = c.dec8(c.A); c.cycles += 4

	// ── INC / DEC 16-bit ─────────────────────────────────────────────────────
	case 0x03: c.SetBC(c.BC() + 1); c.cycles += 8
	case 0x13: c.SetDE(c.DE() + 1); c.cycles += 8
	case 0x23: c.SetHL(c.HL() + 1); c.cycles += 8
	case 0x33: c.SP++; c.cycles += 8
	case 0x0B: c.SetBC(c.BC() - 1); c.cycles += 8
	case 0x1B: c.SetDE(c.DE() - 1); c.cycles += 8
	case 0x2B: c.SetHL(c.HL() - 1); c.cycles += 8
	case 0x3B: c.SP--; c.cycles += 8

	// ADD HL, rr
	case 0x09: c.addHL(c.BC()); c.cycles += 8
	case 0x19: c.addHL(c.DE()); c.cycles += 8
	case 0x29: c.addHL(c.HL()); c.cycles += 8
	case 0x39: c.addHL(c.SP); c.cycles += 8

	// ADD SP, e
	case 0xE8:
		e := int8(c.fetch())
		res := int32(c.SP) + int32(e)
		c.setFlag(FlagZ, false)
		c.setFlag(FlagN, false)
		c.setFlag(FlagH, int(c.SP&0xF)+int(byte(e)&0xF) > 0xF)
		c.setFlag(FlagC, int(c.SP&0xFF)+int(byte(e)) > 0xFF)
		c.SP = uint16(res)
		c.cycles += 16

	// ── Rotates / Shifts ─────────────────────────────────────────────────────
	case 0x07: c.rlca(); c.cycles += 4
	case 0x0F: c.rrca(); c.cycles += 4
	case 0x17: c.rla(); c.cycles += 4
	case 0x1F: c.rra(); c.cycles += 4

	// ── DAA / CPL / SCF / CCF ────────────────────────────────────────────────
	case 0x27: c.daa(); c.cycles += 4
	case 0x2F: c.A = ^c.A; c.setFlag(FlagN, true); c.setFlag(FlagH, true); c.cycles += 4
	case 0x37: c.setFlag(FlagN, false); c.setFlag(FlagH, false); c.setFlag(FlagC, true); c.cycles += 4
	case 0x3F: c.setFlag(FlagN, false); c.setFlag(FlagH, false); c.setFlag(FlagC, !c.flagC()); c.cycles += 4

	// ── Jumps ────────────────────────────────────────────────────────────────
	case 0xC3: c.PC = c.fetch16(); c.cycles += 16
	case 0xE9: c.PC = c.HL(); c.cycles += 4
	case 0xC2: c.jumpCond(!c.flagZ())
	case 0xCA: c.jumpCond(c.flagZ())
	case 0xD2: c.jumpCond(!c.flagC())
	case 0xDA: c.jumpCond(c.flagC())
	case 0x18: c.jr(true)
	case 0x20: c.jr(!c.flagZ())
	case 0x28: c.jr(c.flagZ())
	case 0x30: c.jr(!c.flagC())
	case 0x38: c.jr(c.flagC())

	// ── Calls / Returns ───────────────────────────────────────────────────────
	case 0xCD: c.callNN(); c.cycles += 24
	case 0xC4: c.callCond(!c.flagZ())
	case 0xCC: c.callCond(c.flagZ())
	case 0xD4: c.callCond(!c.flagC())
	case 0xDC: c.callCond(c.flagC())
	case 0xC9: c.PC = c.pop(); c.cycles += 16
	case 0xC0: c.retCond(!c.flagZ())
	case 0xC8: c.retCond(c.flagZ())
	case 0xD0: c.retCond(!c.flagC())
	case 0xD8: c.retCond(c.flagC())
	case 0xD9: c.PC = c.pop(); c.IME = true; c.cycles += 16 // RETI

	// RST
	case 0xC7: c.push(c.PC); c.PC = 0x00; c.cycles += 16
	case 0xCF: c.push(c.PC); c.PC = 0x08; c.cycles += 16
	case 0xD7: c.push(c.PC); c.PC = 0x10; c.cycles += 16
	case 0xDF: c.push(c.PC); c.PC = 0x18; c.cycles += 16
	case 0xE7: c.push(c.PC); c.PC = 0x20; c.cycles += 16
	case 0xEF: c.push(c.PC); c.PC = 0x28; c.cycles += 16
	case 0xF7: c.push(c.PC); c.PC = 0x30; c.cycles += 16
	case 0xFF: c.push(c.PC); c.PC = 0x38; c.cycles += 16

	// ── CB prefix ────────────────────────────────────────────────────────────
	case 0xCB:
		cb := c.fetch()
		c.executeCB(cb)

	default:
		// Unimplemented opcode — treat as NOP
		c.cycles += 4
	}
}

func (c *CPU) jumpCond(cond bool) {
	addr := c.fetch16()
	if cond {
		c.PC = addr
		c.cycles += 16
	} else {
		c.cycles += 12
	}
}

func (c *CPU) jr(cond bool) {
	offset := int8(c.fetch())
	if cond {
		c.PC = uint16(int(c.PC) + int(offset))
		c.cycles += 12
	} else {
		c.cycles += 8
	}
}

func (c *CPU) callNN() {
	addr := c.fetch16()
	c.push(c.PC)
	c.PC = addr
}

func (c *CPU) callCond(cond bool) {
	addr := c.fetch16()
	if cond {
		c.push(c.PC)
		c.PC = addr
		c.cycles += 24
	} else {
		c.cycles += 12
	}
}

func (c *CPU) retCond(cond bool) {
	if cond {
		c.PC = c.pop()
		c.cycles += 20
	} else {
		c.cycles += 8
	}
}

func (c *CPU) rlca() {
	bit7 := c.A >> 7
	c.A = (c.A << 1) | bit7
	c.setFlag(FlagZ, false); c.setFlag(FlagN, false); c.setFlag(FlagH, false)
	c.setFlag(FlagC, bit7 == 1)
}

func (c *CPU) rrca() {
	bit0 := c.A & 1
	c.A = (c.A >> 1) | (bit0 << 7)
	c.setFlag(FlagZ, false); c.setFlag(FlagN, false); c.setFlag(FlagH, false)
	c.setFlag(FlagC, bit0 == 1)
}

func (c *CPU) rla() {
	carry := byte(0)
	if c.flagC() { carry = 1 }
	bit7 := c.A >> 7
	c.A = (c.A << 1) | carry
	c.setFlag(FlagZ, false); c.setFlag(FlagN, false); c.setFlag(FlagH, false)
	c.setFlag(FlagC, bit7 == 1)
}

func (c *CPU) rra() {
	carry := byte(0)
	if c.flagC() { carry = 0x80 }
	bit0 := c.A & 1
	c.A = (c.A >> 1) | carry
	c.setFlag(FlagZ, false); c.setFlag(FlagN, false); c.setFlag(FlagH, false)
	c.setFlag(FlagC, bit0 == 1)
}

func (c *CPU) daa() {
	a := c.A
	if !c.flagN() {
		if c.flagH() || a&0xF > 9 { a += 0x06 }
		if c.flagC() || a > 0x9F { a += 0x60 }
	} else {
		if c.flagH() { a -= 0x06 }
		if c.flagC() { a -= 0x60 }
	}
	c.setFlag(FlagH, false)
	c.setFlag(FlagZ, a == 0)
	if c.A>>7 == 1 { c.setFlag(FlagC, true) }
	c.A = a
}
