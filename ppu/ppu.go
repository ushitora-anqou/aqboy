package ppu

import (
	"github.com/ushitora-anqou/aqboy/bus"
)

const LCD_WIDTH = 160
const LCD_HEIGHT = 144
const BG_PX_WIDTH = 256
const BG_PX_HEIGHT = 256

type PPU struct {
	bus                                             *bus.Bus
	vram                                            [0x2000]uint8
	scx, scy, bgp, lcdc, ly, lyc, wx, wy, wly, stat uint8
	tick                                            uint
}

func NewPPU(bus *bus.Bus) *PPU {
	ppu := &PPU{
		bus: bus,
	}
	ppu.SetMode(2)
	return ppu
}

func (ppu *PPU) Get8(addr uint16) uint8 {
	return ppu.vram[addr-0x8000]
}

func (ppu *PPU) Set8(addr uint16, val uint8) {
	ppu.vram[addr-0x8000] = val
}

func (ppu *PPU) LCDC() uint8 {
	return ppu.lcdc
}

func (ppu *PPU) SetLCDC(lcdc uint8) {
	ppu.lcdc = lcdc
}

func (ppu *PPU) LY() uint8 {
	return ppu.ly
}

func (ppu *PPU) LYC() uint8 {
	return ppu.lyc
}

func (ppu *PPU) SetLYC(lyc uint8) {
	ppu.lyc = lyc
}

func (ppu *PPU) SCX() uint8 {
	return ppu.scx
}

func (ppu *PPU) SetSCX(scx uint8) {
	ppu.scx = scx
}

func (ppu *PPU) SCY() uint8 {
	return ppu.scy
}

func (ppu *PPU) SetSCY(scy uint8) {
	ppu.scy = scy
}

func (ppu *PPU) BGP() uint8 {
	return ppu.bgp
}

func (ppu *PPU) SetBGP(bgp uint8) {
	ppu.bgp = bgp
}

func (ppu *PPU) WY() uint8 {
	return ppu.wy
}

func (ppu *PPU) WX() uint8 {
	return ppu.wx
}

func (ppu *PPU) SetWY(wy uint8) {
	ppu.wy = wy
}

func (ppu *PPU) SetWX(wx uint8) {
	ppu.wx = wx
}

func (ppu *PPU) STAT() uint8 {
	return ppu.stat
}

func (ppu *PPU) SetSTAT(stat uint8) {
	ppu.stat = stat
}

func (ppu *PPU) Mode() uint8 {
	return ppu.stat & 3
}

func (ppu *PPU) SetMode(mode uint8) {
	ppu.stat = (ppu.stat & 0xfc) | mode
}

func (ppu *PPU) getLCDDisplayEnable() bool {
	return (ppu.LCDC()>>7)&1 != 0
}

func (ppu *PPU) getWindowTileMapAddr() uint16 {
	if (ppu.LCDC()>>6)&1 == 0 {
		return 0x9800
	} else {
		return 0x9c00
	}
}

func (ppu *PPU) getWindowDisplayEnable() bool {
	return (ppu.LCDC()>>5)&1 != 0
}

func (ppu *PPU) getBGWindowTileDataArea() bool {
	return (ppu.LCDC()>>4)&1 != 0
}

func (ppu *PPU) getBGTileMapAddr() uint16 {
	if (ppu.LCDC()>>3)&1 == 0 {
		return 0x9800
	} else {
		return 0x9c00
	}
}

func (ppu *PPU) getOBJSize() bool {
	return (ppu.LCDC()>>2)&1 != 0
}

func (ppu *PPU) getOBJDisplayEnable() bool {
	return (ppu.LCDC()>>1)&1 != 0
}

func (ppu *PPU) getBGWindowDisplayPriority() bool {
	return (ppu.LCDC()>>0)&1 != 0
}

func (ppu *PPU) fetchTileColor(isBG bool, x, y int) uint8 {
	tile_x, tile_y := x/8, y/8
	pix_x, pix_y := x%8, y%8

	var tileMapAddr uint16
	if isBG {
		tileMapAddr = ppu.getBGTileMapAddr()
	} else {
		tileMapAddr = ppu.getWindowTileMapAddr()
	}
	tileNo := ppu.Get8(uint16(int(tileMapAddr) + 32*tile_y + tile_x))

	var off uint16
	if ppu.getBGWindowTileDataArea() {
		off = uint16(0x8000 + int(tileNo)*16 + 2*pix_y)
	} else {
		off = uint16(0x9000 + int(int8(tileNo))*16 + 2*pix_y)
	}

	paletteIdxLSB := (ppu.Get8(off) >> (7 - pix_x)) & 1
	paletteIdxMSB := (ppu.Get8(off+1) >> (7 - pix_x)) & 1
	paletteIdx := paletteIdxLSB | (paletteIdxMSB << 1)
	color := (ppu.BGP() >> (2 * paletteIdx)) & 3
	return color
}

func (ppu *PPU) drawLineBG(scanline []uint8) {
	y := int(ppu.ly + ppu.scy) // NOTE: wrap around
	for ax := 0; ax < LCD_WIDTH; ax++ {
		x := int(uint8(ax) + ppu.scx) // NOTE: wrap around
		scanline[ax] = ppu.fetchTileColor(true, x, y)
	}
}

func (ppu *PPU) drawLineWindow(scanline []uint8) {
	if !ppu.getWindowDisplayEnable() {
		return
	}
	y, wx, wy := int(ppu.LY()), int(ppu.WX()-7), int(ppu.WY())
	for x := 0; x < LCD_WIDTH; x++ {
		if x < wx || y < wy {
			continue
		}
		scanline[x] = ppu.fetchTileColor(false, x-wx, int(ppu.wly))
	}
}

func (ppu *PPU) drawLine() error {
	if !ppu.getLCDDisplayEnable() {
		return nil
	}

	scanline := [LCD_WIDTH]uint8{}
	if ppu.getBGWindowDisplayPriority() {
		ppu.drawLineBG(scanline[:])
		ppu.drawLineWindow(scanline[:])
	}
	ppu.bus.LCD.DrawLine(int(ppu.ly), scanline[:])
	return nil
}

func (ppu *PPU) updateInterrupt() {
	cpu := ppu.bus.CPU

	// LCD STAT
	if (cpu.IE()&(1<<1)) != 0 &&
		(((ppu.Mode() == 0) && (ppu.STAT()&(1<<3)) != 0) /* H-Blank */ ||
			((ppu.Mode() == 1) && (ppu.STAT()&(1<<4)) != 0) /* V-Blank */ ||
			((ppu.Mode() == 2) && (ppu.STAT()&(1<<5)) != 0) /* OAM Search */ ||
			(ppu.LY() == ppu.LYC() && (ppu.STAT()&(1<<6)) != 0) /* LY=LYC */) {
		cpu.SetIF(cpu.IF() | (1 << 1))
	}

	// V-Blank
	if (cpu.IE()&(1<<0)) != 0 && ppu.Mode() == 1 {
		cpu.SetIF(cpu.IF() | (1 << 0))
	}
}

func (ppu *PPU) updateLYCLYCoincidence() {
	if ppu.LY() == ppu.LYC() {
		ppu.SetSTAT(ppu.STAT() | (1 << 2))
	} else {
		ppu.SetSTAT(ppu.STAT() &^ (1 << 2))
	}
}

func (ppu *PPU) Update(elapsedTick uint) error {
	ppu.tick += elapsedTick

	switch {
	case ppu.Mode() == 2 && ppu.tick >= 80: // OAM Search --> Pixel Transfer
		ppu.tick -= 80
		ppu.SetMode(3)

	case ppu.Mode() == 3 && ppu.tick >= 168: // Pixel Transfer --> H-Blank
		ppu.tick -= 168
		ppu.SetMode(0)
		ppu.updateInterrupt()

		ppu.drawLine()
		if ppu.WX()-7 < LCD_WIDTH && ppu.WY() <= ppu.LY() {
			ppu.wly += 1
		}

	case ppu.Mode() == 0 && ppu.tick >= 208: // H-Blank --> OAM Search | V-Blank
		ppu.tick -= 208
		ppu.ly += 1
		if ppu.LY() >= 144 { // Go to V-Blank
			ppu.SetMode(1)
		} else { // Go to OAM Search
			ppu.SetMode(2)
		}
		ppu.updateLYCLYCoincidence()
		ppu.updateInterrupt()

	case ppu.Mode() == 1 && ppu.tick >= 4560: // V-Blank --> OAM Search
		ppu.tick -= 4560
		ppu.ly = 0
		ppu.wly = 0
		ppu.SetMode(2)
		ppu.updateLYCLYCoincidence()
		ppu.updateInterrupt()
	}

	return nil
}
