package mmu

import (
	"fmt"
	"os"
)

type Catridge struct {
	rom, ram []uint8
}

func NewCatridge(filePath string) (*Catridge, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// assert Catridge Type == MBC1
	if src[0x147] != 0x01 {
		return nil, fmt.Errorf("Unsupported Catridge Type: %d", src[0x147])
	}
	// assert ROM Size == 32KByte
	if src[0x148] != 0x00 {
		return nil, fmt.Errorf("Unsupported ROM Size: %d", src[0x148])
	}
	// assert RAM Size == None
	if src[0x149] != 0x00 {
		return nil, fmt.Errorf("Unsupported RAM Size: %d", src[0x149])
	}

	return &Catridge{
		rom: src,
		ram: []uint8{},
	}, nil
}
