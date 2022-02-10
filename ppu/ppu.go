package ppu

import (
	"sort"

	"github.com/ushitora-anqou/aqboy/bus"
	"github.com/ushitora-anqou/aqboy/constant"
)

type PPU struct {
	bus                                                         *bus.Bus
	vram                                                        [0x2000]uint8
	oam                                                         [0xa0]uint8
	scx, scy, bgp, obp0, obp1, lcdc, ly, lyc, wx, wy, wly, stat uint8
	tick                                                        uint
}

func NewPPU(bus *bus.Bus) *PPU {
	ppu := &PPU{
		bus: bus,
	}
	ppu.SetMode(2)
	return ppu
}

func (ppu *PPU) GetVRAM8(index uint16) uint8 {
	return ppu.vram[index]
}

func (ppu *PPU) SetVRAM8(index uint16, val uint8) {
	ppu.vram[index] = val
}

func (ppu *PPU) GetOAM8(index uint16) uint8 {
	return ppu.oam[index]
}

func (ppu *PPU) SetOAM8(index uint16, val uint8) {
	ppu.oam[index] = val
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

func (ppu *PPU) OBP0() uint8 {
	return ppu.obp0
}

func (ppu *PPU) SetOBP0(obp0 uint8) {
	ppu.obp0 = obp0
}

func (ppu *PPU) OBP1() uint8 {
	return ppu.obp1
}

func (ppu *PPU) SetOBP1(obp1 uint8) {
	ppu.obp1 = obp1
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

func (ppu *PPU) getOBJYSize() uint8 {
	if (ppu.LCDC()>>2)&1 == 0 {
		return 8
	} else {
		return 16
	}
}

func (ppu *PPU) getOBJDisplayEnable() bool {
	return (ppu.LCDC()>>1)&1 != 0
}

func (ppu *PPU) getBGWindowDisplayPriority() bool {
	return (ppu.LCDC()>>0)&1 != 0
}

func (ppu *PPU) fetchTileIndex(isBG bool, x, y int) uint8 {
	tile_x, tile_y := x/8, y/8

	var tileMapAddr uint16
	if isBG {
		tileMapAddr = ppu.getBGTileMapAddr()
	} else {
		tileMapAddr = ppu.getWindowTileMapAddr()
	}

	tileNo := ppu.GetVRAM8(uint16(int(tileMapAddr) - 0x8000 + 32*tile_y + tile_x))
	return tileNo
}

func (ppu *PPU) fetchTileColor(isObject bool, tileNo, paletteData uint8, pixX, pixY int) (uint8, uint8) {
	var off uint16
	switch {
	case isObject:
		if ppu.getOBJYSize() == 16 {
			// Bit 0 of tile index for 8x16 objects should be ignored.
			tileNo &^= 1 << 0
		}
		off = uint16(int(tileNo)*16 + 2*pixY)
	case ppu.getBGWindowTileDataArea():
		off = uint16(int(tileNo)*16 + 2*pixY)
	default:
		off = uint16(0x1000 + int(int8(tileNo))*16 + 2*pixY)
	}

	paletteIdxLSB := (ppu.GetVRAM8(off) >> (7 - pixX)) & 1
	paletteIdxMSB := (ppu.GetVRAM8(off+1) >> (7 - pixX)) & 1
	paletteIdx := paletteIdxLSB | (paletteIdxMSB << 1)
	color := (paletteData >> (2 * paletteIdx)) & 3
	return paletteIdx, color
}

func (ppu *PPU) fetchBGWindowTileColor(isBG bool, x, y int) uint8 {
	tileNo := ppu.fetchTileIndex(isBG, x, y)
	_, color := ppu.fetchTileColor(false, tileNo, ppu.BGP(), x%8, y%8)
	return color
}

func (ppu *PPU) drawLineBG(scanline []uint8) {
	y := int(ppu.ly + ppu.scy) // NOTE: wrap around
	for ax := 0; ax < constant.LCD_WIDTH; ax++ {
		x := int(uint8(ax) + ppu.scx) // NOTE: wrap around
		scanline[ax] = ppu.fetchBGWindowTileColor(true, x, y)
	}
}

func (ppu *PPU) drawLineWindow(scanline []uint8) {
	if !ppu.getWindowDisplayEnable() {
		return
	}
	y, wx, wy := int(ppu.LY()), int(ppu.WX()-7), int(ppu.WY())
	for x := 0; x < constant.LCD_WIDTH; x++ {
		if x < wx || y < wy {
			continue
		}
		scanline[x] = ppu.fetchBGWindowTileColor(false, x-wx, int(ppu.wly))
	}
}

func (ppu *PPU) drawLineObjects(scanline []uint8) {
	objXSize := 8
	objYSize := int(ppu.getOBJYSize())
	ly := int(ppu.LY())

	// Select objects to be drawn
	objs := []*object{}
	for i := 0; i < 40 && len(objs) < 10; i++ {
		obj := newObject(ppu.bus.MMU, uint16(0xfe00+i*4))
		if obj.screenY() <= ly && ly < obj.screenY()+objYSize {
			objs = append(objs, obj)
		}
	}

	// Sort the selected objects IN REVERSE
	sort.Sort(sort.Reverse(byXAndOAMIndex(objs)))

	// Render the objects in order
	for _, obj := range objs {
		for ax := 0; ax < objXSize; ax++ {
			paletteData := ppu.OBP0()
			if obj.paletteNumber() {
				paletteData = ppu.OBP1()
			}

			oy := int(ly - obj.screenY())
			if obj.yFlip() {
				oy = (objYSize - 1) - oy
			}
			ox := ax
			if obj.xFlip() {
				ox = (objXSize - 1) - ox
			}

			paletteIdx, color := ppu.fetchTileColor(true, obj.tileIndex, paletteData, ox, oy)

			x := int(obj.screenX()) + ax
			if 0 <= x && x < constant.LCD_WIDTH && paletteIdx != 0 /* transparent */ {
				scanline[x] = color
			}
		}
	}
}

func (ppu *PPU) drawLine() error {
	if !ppu.getLCDDisplayEnable() {
		return nil
	}

	scanline := [constant.LCD_WIDTH]uint8{}
	if ppu.getBGWindowDisplayPriority() {
		ppu.drawLineBG(scanline[:])
		ppu.drawLineWindow(scanline[:])
	}
	if ppu.getOBJDisplayEnable() {
		ppu.drawLineObjects(scanline[:])
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
		if ppu.WX()-7 < constant.LCD_WIDTH && ppu.WY() <= ppu.LY() {
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

	case ppu.Mode() == 1 && ppu.tick >= 456: // V-Blank --> V-Blank | OAM Search
		ppu.tick -= 456
		ppu.ly += 1
		if ppu.LY() == 155 { // V-Blank --> OAM Search
			ppu.ly = 0
			ppu.wly = 0
			ppu.SetMode(2)
			ppu.updateInterrupt()
		}
		ppu.updateLYCLYCoincidence()
	}

	return nil
}

func (ppu *PPU) TransferOAM(srcPrefix uint8) {
	copy(ppu.oam[:], ppu.bus.MMU.GetSliceXX00(int(srcPrefix), 0xa0))
}
