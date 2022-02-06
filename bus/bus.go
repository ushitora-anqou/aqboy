package bus

import (
	"log"
)

type InterruptBits struct {
	vblank, lcd, timer, serial, joypad bool
}

func (ib *InterruptBits) getN(i int) bool {
	switch i {
	case 0:
		return ib.vblank
	case 1:
		return ib.lcd
	case 2:
		return ib.timer
	case 3:
		return ib.serial
	case 4:
		return ib.joypad
	default:
		log.Fatalf("Invalid interrupt bit: %d", i)
	}
	return false
}

func (ib *InterruptBits) setN(i int, val bool) {
	switch i {
	case 0:
		ib.vblank = val
	case 1:
		ib.lcd = val
	case 2:
		ib.timer = val
	case 3:
		ib.serial = val
	case 4:
		ib.joypad = val
	default:
		log.Fatalf("Invalid interrupt bit: %d", i)
	}
}

func (ib *InterruptBits) get() uint8 {
	var ret uint8 = 0
	for i := 0; i < 5; i++ {
		if ib.getN(i) {
			ret |= 1 << i
		}
	}
	return ret
}

func (ib *InterruptBits) set(val uint8) {
	for i := 0; i < 5; i++ {
		ib.setN(i, ((val>>i)&1) != 0)
	}
}

type CPU interface {
	SetIE(val uint8)
	SetIF(val uint8)
	IE() uint8
	IF() uint8
}

type MMU interface {
	Get8(addr uint16) uint8
	Get16(addr uint16) uint16
	Set8(addr uint16, val uint8)
	Set16(addr uint16, val uint16)
}

type PPU interface {
	Get8(addr uint16) uint8
	Set8(addr uint16, val uint8)

	LCDC() uint8
	LY() uint8
	SCX() uint8
	SCY() uint8
	BGP() uint8

	SetLCDC(lcdc uint8)
	SetSTAT(stat uint8)
	SetSCX(scx uint8)
	SetSCY(scy uint8)
	SetBGP(bgp uint8)
	SetWX(wx uint8)
	SetWY(wy uint8)
	SetLYC(lyc uint8)
}

type LCD interface {
	DrawLine(ly int, scanline []uint8) error
}

type Timer interface {
	TIMA() uint8
	TMA() uint8
	TAC() uint8
	SetTIMA(val uint8)
	SetTMA(val uint8)
	SetTAC(val uint8)
}

type Bus struct {
	CPU
	MMU
	PPU
	LCD
	Timer
}

func NewBus() *Bus {
	return &Bus{}
}

func (b *Bus) Register(cpu CPU, mmu MMU, ppu PPU, lcd LCD, timer Timer) {
	b.CPU = cpu
	b.MMU = mmu
	b.PPU = ppu
	b.LCD = lcd
	b.Timer = timer
}
