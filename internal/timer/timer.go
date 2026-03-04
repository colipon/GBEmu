package timer

import "github.com/colipon/gbemu/internal/mmu"

// Timer implements the Game Boy timer hardware
type Timer struct {
	mem       *mmu.MMU
	divCycles int
	timCycles int
}

// New creates a Timer
func New(mem *mmu.MMU) *Timer {
	t := &Timer{mem: mem}
	mem.OnTimerWrite = t.WriteCallback
	return t
}

// tick frequencies indexed by TAC bits 0-1
var tacFreqs = [4]int{1024, 16, 64, 256}

// Step advances the timer by the given number of T-cycles
func (t *Timer) Step(cycles int) {
	// DIV increments every 256 T-cycles
	t.divCycles += cycles
	for t.divCycles >= 256 {
		t.divCycles -= 256
		t.mem.IO[0x04]++ // DIV register
	}

	tac := t.mem.IO[0x07]
	if tac&0x04 == 0 {
		return // Timer disabled
	}

	freq := tacFreqs[tac&0x03]
	t.timCycles += cycles
	for t.timCycles >= freq {
		t.timCycles -= freq
		tima := t.mem.IO[0x05]
		if tima == 0xFF {
			t.mem.IO[0x05] = t.mem.IO[0x06] // reload TMA
			t.mem.RequestInterrupt(mmu.IntTimer)
		} else {
			t.mem.IO[0x05]++
		}
	}
}

// WriteCallback resets DIV on write to FF04
func (t *Timer) WriteCallback(addr uint16, _ byte) {
	if addr == 0xFF04 {
		t.mem.IO[0x04] = 0
		t.divCycles = 0
	}
}
