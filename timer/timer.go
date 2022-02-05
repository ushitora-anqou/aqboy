package timer

import (
	"github.com/ushitora-anqou/aqboy/bus"
)

type Timer struct {
	bus            *bus.Bus
	tima, tma, tac uint8
	tick           uint
}

func NewTimer(bus *bus.Bus) *Timer {
	return &Timer{
		bus: bus,
	}
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

	sel := t.inputClockSelect()
	t.tick += tick
	var thr uint
	switch {
	case sel == 0: // CPU Clock / 1024
		thr = 1024
	case sel == 1: // CPU Clock / 16
		thr = 16
	case sel == 2: // CPU Clock / 64
		thr = 64
	case sel == 3: // CPU Clock / 256
		thr = 256
	}
	for t.tick > thr {
		t.tick -= thr
		t.incTIMA()
	}
}
