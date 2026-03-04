package cpu

// executeCB handles all 0xCB prefixed instructions
func (c *CPU) executeCB(op byte) {
	reg := op & 0x07
	bit := (op >> 3) & 0x07

	getR := func() byte {
		switch reg {
		case 0: return c.B
		case 1: return c.C
		case 2: return c.D
		case 3: return c.E
		case 4: return c.H
		case 5: return c.L
		case 6: return c.mem.Read(c.HL())
		case 7: return c.A
		}
		return 0
	}
	setR := func(v byte) {
		switch reg {
		case 0: c.B = v
		case 1: c.C = v
		case 2: c.D = v
		case 3: c.E = v
		case 4: c.H = v
		case 5: c.L = v
		case 6: c.mem.Write(c.HL(), v)
		case 7: c.A = v
		}
	}
	hlCycles := func(base int) int {
		if reg == 6 { return base + 8 }
		return base
	}

	switch op >> 6 {
	case 0: // Rotates/Shifts
		val := getR()
		var res byte
		switch (op >> 3) & 0x07 {
		case 0: // RLC
			res = (val << 1) | (val >> 7)
			c.setFlag(FlagC, val>>7 == 1)
		case 1: // RRC
			res = (val >> 1) | (val << 7)
			c.setFlag(FlagC, val&1 == 1)
		case 2: // RL
			carry := byte(0)
			if c.flagC() { carry = 1 }
			c.setFlag(FlagC, val>>7 == 1)
			res = (val << 1) | carry
		case 3: // RR
			carry := byte(0)
			if c.flagC() { carry = 0x80 }
			c.setFlag(FlagC, val&1 == 1)
			res = (val >> 1) | carry
		case 4: // SLA
			c.setFlag(FlagC, val>>7 == 1)
			res = val << 1
		case 5: // SRA
			c.setFlag(FlagC, val&1 == 1)
			res = (val >> 1) | (val & 0x80)
		case 6: // SWAP
			res = (val << 4) | (val >> 4)
			c.setFlag(FlagC, false)
		case 7: // SRL
			c.setFlag(FlagC, val&1 == 1)
			res = val >> 1
		}
		c.setFlag(FlagZ, res == 0)
		c.setFlag(FlagN, false)
		c.setFlag(FlagH, false)
		setR(res)
		c.cycles += hlCycles(8)

	case 1: // BIT
		val := getR()
		c.setFlag(FlagZ, val>>bit&1 == 0)
		c.setFlag(FlagN, false)
		c.setFlag(FlagH, true)
		c.cycles += hlCycles(8)

	case 2: // RES
		val := getR() &^ (1 << bit)
		setR(val)
		c.cycles += hlCycles(8)

	case 3: // SET
		val := getR() | (1 << bit)
		setR(val)
		c.cycles += hlCycles(8)
	}
}
