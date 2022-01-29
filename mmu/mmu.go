package mmu

import "log"

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
	cat              *Catridge
	vram, wram, hram []uint8
}

func NewMMU(catridgeFilePath string) (*MMU, error) {
	cat, err := NewCatridge(catridgeFilePath)
	if err != nil {
		return nil, err
	}
	mmu := &MMU{
		cat:  cat,
		vram: make([]uint8, 0x2000),
		wram: make([]uint8, 0x2000),
		hram: make([]uint8, 0x007f),
	}
	return mmu, nil
}

func (mmu *MMU) Set8(addr uint16, val uint8) {
	switch {
	case 0x8000 <= addr && addr <= 0x9FFF:
		mmu.vram[addr-0x8000] = val
	case 0xc000 <= addr && addr <= 0xdfff:
		mmu.wram[addr-0xc000] = val
	case 0xe000 <= addr && addr <= 0xfdff:
		mmu.wram[addr-0xe000] = val
	case 0xff80 <= addr && addr <= 0xfffe:
		mmu.hram[addr-0xff80] = val
	case addr == 0xff07:
		timerEnable := (val >> 2) & 1
		inputClockSelect := val & 3
		log.Printf("\t<<<WRITE: TAC Timer Control: %v %v>>>", timerEnable, inputClockSelect)
	case addr == 0xff26:
		allSoundOnOff := (val >> 7) & 1
		soundOnFlag := val & 15
		log.Printf("\t<<<WRITE: NR52 Sound on/off: %v %b>>>", allSoundOnOff, soundOnFlag)
	case addr == 0xff0f:
		vBlank := val & 1
		lcdStat := (val >> 1) & 1
		timer := (val >> 2) & 1
		serial := (val >> 3) & 1
		joypad := (val >> 4) & 1
		log.Printf("\t<<<WRITE: IF Interrupt Flag: %v %v %v %v %v>>>", vBlank, lcdStat, timer, serial, joypad)
	case addr == 0xff24:
		outputVinToSO2 := (val >> 7) & 1
		so2OutputLevel := (val >> 4) & 7
		outputVinToSO1 := (val >> 3) & 1
		so1OutputLevel := (val >> 0) & 7
		log.Printf("\t<<<WRITE: NR50 Channel control / On-OFF / Volume: %v %v %v %v>>>", outputVinToSO2, so2OutputLevel, outputVinToSO1, so1OutputLevel)
	case addr == 0xff25:
		log.Printf("\t<<<WRITE: NR51 Selection of Sound output terminal: %b>>>", val)
	case addr == 0xffff:
		vBlank := val & 1
		lcdStat := (val >> 1) & 1
		timer := (val >> 2) & 1
		serial := (val >> 3) & 1
		joypad := (val >> 4) & 1
		log.Printf("\t<<<WRITE: IE Interrupt Enable: %v %v %v %v %v>>>", vBlank, lcdStat, timer, serial, joypad)
	default:
		log.Fatalf("Invalid memory access of Set8: 0x%02x at 0x%08x", val, addr)
	}
}

func (mmu *MMU) Get8(addr uint16) uint8 {
	switch {
	case 0x0000 <= addr && addr <= 0x3FFF:
		return mmu.cat.rom[addr]
	case 0x4000 <= addr && addr <= 0x7FFF:
		return mmu.cat.rom[addr]
	case 0x8000 <= addr && addr <= 0x9FFF:
		return mmu.vram[addr-0x8000]
	case 0xc000 <= addr && addr <= 0xdfff:
		return mmu.wram[addr-0xc000]
	case 0xe000 <= addr && addr <= 0xfdff:
		return mmu.wram[addr-0xe000]
	case 0xff80 <= addr && addr <= 0xfffe:
		return mmu.hram[addr-0xff80]
	case addr == 0xff07:
		log.Printf("\t<<<READ: TAC Timer Control>>>")
	case addr == 0xff0f:
		log.Printf("\t<<<READ: IF Interrupt Flag>>>")
	case addr == 0xffff:
		log.Printf("\t<<<READ: IF Interrupt Enable>>>")
	case addr == 0xff24:
		log.Printf("\t<<<READ: NR50 Channel control / On-OFF / Volume>>>")
	case addr == 0xff25:
		log.Printf("\t<<<READ: NR51 Selection of Sound output terminal>>>")
	case addr == 0xff26:
		log.Printf("\t<<<READ: NR52 Sound on/off>>>")
	}
	log.Fatalf("Invalid memory access of Get8: at 0x%08x", addr)
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
