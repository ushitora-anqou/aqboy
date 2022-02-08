package mmu

type Cartridge interface {
	get8(addr uint16) uint8
	set8(addr uint16, val uint8)
}
