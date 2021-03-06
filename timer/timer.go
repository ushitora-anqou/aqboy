package timer

import (
	"github.com/ushitora-anqou/aqboy/bus"
)

type Timer struct {
	bus                 *bus.Bus
	tima, tma, tac, div uint8
	tick, tickDiv       uint
}

func NewTimer(bus *bus.Bus) *Timer {
	return &Timer{
		bus: bus,
	}
}

func (t *Timer) DIV() uint8 {
	return t.div
}

func (t *Timer) TIMA() uint8 {
	return t.tima
}

func (t *Timer) TMA() uint8 {
	return t.tma
}

func (t *Timer) TAC() uint8 {
	return t.tac
}

func (t *Timer) ResetDIV() {
	t.div = 0
}

func (t *Timer) SetTIMA(val uint8) {
	t.tima = val
}

func (t *Timer) SetTMA(val uint8) {
	t.tma = val
}

func (t *Timer) SetTAC(val uint8) {
	t.tac = val
}

func (t *Timer) incTIMA() {
	val := uint(t.TIMA()) + 1
	if val > 0xff { // Interrupt
		cpu := t.bus.CPU
		cpu.SetIF(cpu.IF() | (1 << 2))
		val = uint(t.TMA())
	}
	t.SetTIMA(uint8(val))
}

func (t *Timer) timerEnable() bool {
	return ((t.TAC() >> 2) & 1) != 0
}

func (t *Timer) inputClockSelect() uint8 {
	return t.TAC() & 3
}

func (t *Timer) Update(tick uint) {
	if !t.timerEnable() {
		return
	}

	t.tick += tick
	var thr uint
	switch t.inputClockSelect() {
	case 0: // CPU Clock / 1024
		thr = 1024
	case 1: // CPU Clock / 16
		thr = 16
	case 2: // CPU Clock / 64
		thr = 64
	case 3: // CPU Clock / 256
		thr = 256
	}
	for t.tick > thr {
		t.tick -= thr
		t.incTIMA()
	}

	t.tickDiv += tick
	if t.tickDiv > 16384 {
		t.tickDiv -= 16384
		t.div += 1
	}
}
