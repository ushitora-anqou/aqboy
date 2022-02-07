package mmu

import (
	"log"

	"github.com/ushitora-anqou/aqboy/bus"
)

func dbgpr(format string, v ...interface{}) {
	log.Printf(format, v...)
}

type MMU struct {
	/*
		GENERAL MEMORY MAP
		Thanks to: https://gbdev.gg8.se/wiki/articles/Memory_Map

		0000-3FFF  16KB ROM bank 00 	From cartridge
		4000-7FFF  16KB ROM Bank 01-NN 	From cartridge
		8000-9FFF  8KB Video RAM (VRAM)
		A000-BFFF  8KB External RAM     In cartridge
		C000-CFFF  4KB Work RAM (WRAM)
		D000-DFFF  4KB Work RAM (WRAM)
		E000-FDFF  Mirror of C000-DDFF (ECHO RAM)
		FE00-FE9F  Sprite attribute table (OAM)
		FEA0-FEFF  Not Usable
		FF00-FF7F  I/O Registers
		FF80-FFFE  High RAM (HRAM)
		FFFF-FFFF  Interrupts Enable Register (IE)
	*/
	bus        *bus.Bus
	cat        *Catridge
	wram, hram []uint8
	oam        [0x00a0]uint8
}

func NewMMU(bus *bus.Bus, catridgeFilePath string) (*MMU, error) {
	cat, err := NewCatridge(catridgeFilePath)
	if err != nil {
		return nil, err
	}
	mmu := &MMU{
		bus:  bus,
		cat:  cat,
		wram: make([]uint8, 0x2000),
		hram: make([]uint8, 0x007f),
	}
	return mmu, nil
}

func (mmu *MMU) Set8(addr uint16, val uint8) {
	cpu := mmu.bus.CPU
	ppu := mmu.bus.PPU
	timer := mmu.bus.Timer

	switch {
	case 0x8000 <= addr && addr <= 0x9FFF:
		ppu.Set8(addr, val)
		return
	case 0xc000 <= addr && addr <= 0xdfff:
		mmu.wram[addr-0xc000] = val
		return
	case 0xe000 <= addr && addr <= 0xfdff:
		mmu.wram[addr-0xe000] = val
		return
	case 0xfe00 <= addr && addr <= 0xfe9f:
		mmu.oam[addr-0xfe00] = val
		return
	case 0xfea0 <= addr && addr <= 0xfeff:
		log.Fatalf("Invalid memory access of Get8: at 0x%08x", addr)
	case 0xff80 <= addr && addr <= 0xfffe:
		mmu.hram[addr-0xff80] = val
		return
	}

	switch addr {
	case 0xff00:
		dbgpr("\t<<<WRITE: P1/JOYP Joypad: %08b>>>", val)
	case 0xff01:
		dbgpr("\t<<<WRITE: SB Serial transfer data>>>")
	case 0xff02:
		dbgpr("\t<<<WRITE: SC Serial Transfer Control>>>")
	case 0xff05:
		dbgpr("\t<<<WRITE: TIMA Timer counter: %02x>>>", val)
		timer.SetTIMA(val)
	case 0xff06:
		dbgpr("\t<<<WRITE: TMA Timer Modulo: %02x>>>", val)
		timer.SetTMA(val)
	case 0xff07:
		dbgpr("\t<<<WRITE: TAC Timer Control: %b>>>", val)
		timer.SetTAC(val)
	case 0xff0f:
		dbgpr("\t<<<WRITE: IF Interrupt Flag: %b>>>", val)
		cpu.SetIF(val)
	case 0xff24:
		outputVinToSO2 := (val >> 7) & 1
		so2OutputLevel := (val >> 4) & 7
		outputVinToSO1 := (val >> 3) & 1
		so1OutputLevel := (val >> 0) & 7
		dbgpr("\t<<<WRITE: NR50 Channel control / On-OFF / Volume: %v %v %v %v>>>", outputVinToSO2, so2OutputLevel, outputVinToSO1, so1OutputLevel)
	case 0xff25:
		dbgpr("\t<<<WRITE: NR51 Selection of Sound output terminal: %08b>>>", val)
	case 0xff26:
		allSoundOnOff := (val >> 7) & 1
		soundOnFlag := val & 15
		dbgpr("\t<<<WRITE: NR52 Sound on/off: %v %08b>>>", allSoundOnOff, soundOnFlag)
	case 0xff40:
		dbgpr("\t<<<WRITE: LCDC LCD Control: %08b>>>", val)
		ppu.SetLCDC(val)
	case 0xff41:
		dbgpr("\t<<<WRITE: STAT LCDC Status: %08b>>>", val)
		ppu.SetSTAT(val)
	case 0xff42:
		dbgpr("\t<<<WRITE: SCY Scroll Y: 0x%02x>>>", val)
		ppu.SetSCY(val)
	case 0xff43:
		dbgpr("\t<<<WRITE: SCX Scroll X: 0x%02x>>>", val)
		ppu.SetSCX(val)
	case 0xff45:
		dbgpr("\t<<<WRITE: LYC LY Compare: 0x%02x>>>", val)
		ppu.SetLYC(val)
	case 0xff47:
		dbgpr("\t<<<WRITE: BGP BG Palette Data Non CGB Mode Only: %08b>>>", val)
		ppu.SetBGP(val)
	case 0xff48:
		dbgpr("\t<<<WRITE: OBP0 Object Palette 0 Data Non CGB Mode Only %08b>>>", val)
		ppu.SetOBP0(val)
	case 0xff49:
		dbgpr("\t<<<WRITE: OBP1 Object Palette 1 Data Non CGB Mode Only %08b>>>", val)
		ppu.SetOBP1(val)
	case 0xff4a:
		dbgpr("\t<<<WRITE: WY Window Y Position: 0x%02x>>>", val)
		mmu.bus.PPU.SetWY(val)
	case 0xff4b:
		dbgpr("\t<<<WRITE: WX Window X Position: 0x%02x>>>", val)
		mmu.bus.PPU.SetWX(val)
	case 0xff4d:
		dbgpr("\t<<<WRITE: KEY1 CGB Mode Only Prepare Speed Switch>>>")
	case 0xff4f:
		dbgpr("\t<<<WRITE: VBK CGB Mode Only VRAM Bank>>>")
	case 0xff68:
		dbgpr("\t<<<WRITE: BCPS/BGPI CGB Mode Only Background Palette Index>>>")
	case 0xff69:
		dbgpr("\t<<<WRITE: BCPD/BGPD CGB Mode Only Background Palette Data>>>")
	case 0xffff:
		dbgpr("\t<<<WRITE: IE Interrupt Enable: %b>>>", val)
		cpu.SetIE(val)
	default:
		log.Fatalf("Invalid memory access of Set8: 0x%02x at 0x%08x", val, addr)
	}
}

func (mmu *MMU) Get8(addr uint16) uint8 {
	ppu := mmu.bus.PPU
	cpu := mmu.bus.CPU
	timer := mmu.bus.Timer

	switch {
	case 0x0000 <= addr && addr <= 0x3FFF:
		return mmu.cat.rom[addr]
	case 0x4000 <= addr && addr <= 0x7FFF:
		return mmu.cat.rom[addr]
	case 0x8000 <= addr && addr <= 0x9FFF:
		return ppu.Get8(addr)
	case 0xc000 <= addr && addr <= 0xdfff:
		return mmu.wram[addr-0xc000]
	case 0xe000 <= addr && addr <= 0xfdff:
		return mmu.wram[addr-0xe000]
	case 0xfe00 <= addr && addr <= 0xfe9f:
		return mmu.oam[addr-0xfe00]
	case 0xfea0 <= addr && addr <= 0xfeff:
		log.Fatalf("Invalid memory access of Get8: at 0x%08x", addr)
	case 0xff80 <= addr && addr <= 0xfffe:
		return mmu.hram[addr-0xff80]
	}

	switch addr {
	case 0xff00:
		dbgpr("\t<<<READ: P1/JOYP Joypad>>>")
	case 0xff05:
		dbgpr("\t<<<READ: TIMA Timer counter>>>")
		return timer.TIMA()
	case 0xff06:
		dbgpr("\t<<<READ: TMA Timer Modulo>>>")
		return timer.TMA()
	case 0xff07:
		dbgpr("\t<<<READ: TAC Timer Control>>>")
		return timer.TAC()
	case 0xff0f:
		dbgpr("\t<<<READ: IF Interrupt Flag>>>")
		return cpu.IF()
	case 0xffff:
		dbgpr("\t<<<READ: IE Interrupt Enable>>>")
		return cpu.IE()
	case 0xff24:
		dbgpr("\t<<<READ: NR50 Channel control / On-OFF / Volume>>>")
	case 0xff25:
		dbgpr("\t<<<READ: NR51 Selection of Sound output terminal>>>")
	case 0xff26:
		dbgpr("\t<<<READ: NR52 Sound on/off>>>")
	case 0xff40:
		dbgpr("\t<<<READ: LCDC LCD Control>>>")
		return ppu.LCDC()
	case 0xff44:
		dbgpr("\t<<<READ: LY - LCDC Y-Coordinate>>>")
		return ppu.LY()
	case 0xff4d:
		dbgpr("\t<<<READ: KEY1 CGB Mode Only Prepare Speed Switch>>>")
	case 0xff68:
		dbgpr("\t<<<READ: BCPS/BGPI CGB Mode Only Background Palette Index>>>")
	default:
		log.Fatalf("Invalid memory access of Get8: at 0x%08x", addr)
	}

	return 0
}

func (mmu *MMU) Get16(addr uint16) uint16 {
	lo := (uint16)(mmu.Get8(addr))
	hi := (uint16)(mmu.Get8(addr + 1))
	return lo + (hi << 8)
}

func (mmu *MMU) Set16(addr uint16, val uint16) {
	mmu.Set8(addr, uint8(val))
	mmu.Set8(addr+1, uint8(val>>8))
}
