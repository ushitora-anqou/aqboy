package mmu

import (
	"fmt"
	"log"
	"os"
)

type MBC1Cartridge struct {
	rom, ram                                               []uint8
	log2ROMBanks, romBankNumber, bankingMode, secondaryReg int
	ramEnabled, largeROM                                   bool
}

func NewMBC1Cartridge(filePath string) (*MBC1Cartridge, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Catridge Type
	catType := src[0x147]
	if catType > 3 {
		return nil, fmt.Errorf("Unsupported Cartridge Type: %d", catType)
	}

	//  ROM Size
	log2ROMBanks := 1 << (src[0x148] + 1)
	if log2ROMBanks > 7 {
		return nil, fmt.Errorf("Unsupported ROM Size: %d", src[0x148])
	}

	// RAM Size
	ramSize := 0
	switch src[0x149] {
	case 0:
		// Do nothing
	case 2:
		ramSize = 8 * 1024
	case 3:
		ramSize = 32 * 1024
	case 4:
		ramSize = 128 * 1024
	case 5:
		ramSize = 64 * 1024
	default:
		return nil, fmt.Errorf("Unsupported RAM Size: %d", src[0x149])
	}

	return &MBC1Cartridge{
		rom:           src,
		ram:           make([]uint8, ramSize),
		log2ROMBanks:  log2ROMBanks,
		romBankNumber: 1,
		bankingMode:   0,
		secondaryReg:  0,
		ramEnabled:    false,
		largeROM:      ramSize <= 8*1024,
	}, nil
}

func (cat *MBC1Cartridge) set8(addr uint16, val uint8) {
	switch {
	case 0x0000 <= addr && addr <= 0x1fff: // RAM Enable
		cat.ramEnabled = val&0x0f == 0x0a

	case 0x2000 <= addr && addr <= 0x3fff: // ROM Bank Number (lower 5 bits)
		width := cat.log2ROMBanks
		if width > 5 {
			width = 5
		}
		num := int(val) & ((1 << width) - 1)
		if num == 0 {
			num = 1
		}
		cat.romBankNumber = num

	case 0x4000 <= addr && addr <= 0x5fff: // RAM Bank Number or Upper Bits of ROM Bank Number (2 bits)
		cat.secondaryReg = int(val & 0x03)

	case 0x6000 <= addr && addr <= 0x7fff: // Banking Mode Select
		cat.bankingMode = int(val & 0x1)

	case 0xa000 <= addr && addr <= 0xbfff: // RAM Bank
		index := cat.getRAMIndex(addr)
		cat.ram[index] = val

	default:
		log.Fatalf("Invalid address")
	}
}

func (cat *MBC1Cartridge) isROMBankingEnabled() bool {
	return cat.bankingMode == 0 || cat.largeROM
}

func (cat *MBC1Cartridge) isRAMBankingEnabled() bool {
	return !cat.isROMBankingEnabled()
}

func (cat *MBC1Cartridge) isUnbankableBank0Enabled() bool {
	return cat.bankingMode != 0 && cat.largeROM
}

func (cat *MBC1Cartridge) getRAMIndex(addr uint16) int {
	index := int(addr - 0xa000)
	if cat.isRAMBankingEnabled() {
		index += cat.secondaryReg * 0x2000
	}
	return index
}

func (cat *MBC1Cartridge) get8(addr uint16) uint8 {
	switch {
	case 0x0000 <= addr && addr <= 0x3fff: // ROM Bank X0
		bank := 0
		if cat.isUnbankableBank0Enabled() {
			bank = cat.secondaryReg << 5
		}
		index := bank*0x4000 + int(addr)
		return cat.rom[index]

	case 0x4000 <= addr && addr <= 0x7fff: // ROM Bank 01-7F
		bank := 0
		if cat.isROMBankingEnabled() {
			bank = (cat.secondaryReg << 5) | cat.romBankNumber
		}
		index := bank*0x4000 + int(addr-0x4000)
		return cat.rom[index]

	case 0xa000 <= addr && addr <= 0xbfff: // RAM Bank
		index := cat.getRAMIndex(addr)
		return cat.ram[index]
	}

	log.Fatalf("Invalid address")
	return 0
}
