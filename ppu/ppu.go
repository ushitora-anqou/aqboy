package ppu

import "github.com/ushitora-anqou/aqboy/bus"

const LCD_WIDTH = 160
const LCD_HEIGHT = 144
const BG_PX_WIDTH = 256
const BG_PX_HEIGHT = 256

type pixelsBuilder struct {
	scx, scy int
	pixels   [LCD_WIDTH * LCD_HEIGHT]uint8
}

func newPixelsBuilder(scx, scy int) *pixelsBuilder {
	return &pixelsBuilder{
		scx, scy, [LCD_WIDTH * LCD_HEIGHT]uint8{},
	}
}

func (b *pixelsBuilder) getPixels() []uint8 {
	return b.pixels[:]
}

func (b *pixelsBuilder) set(srcX, srcY int, pixel uint8) {
	x := srcX - b.scx
	y := srcY - b.scy
	if x < 0 {
		x += BG_PX_WIDTH
	}
	if y < 0 {
		y += BG_PX_HEIGHT
	}
	if x < LCD_WIDTH && y < LCD_HEIGHT {
		b.pixels[y*LCD_WIDTH+x] = pixel
	}
}

type PPU struct {
	bus                           *bus.Bus
	vram                          [0x2000]uint8
	scx, scy, bgp, mode, lcdc, ly uint8
	tick                          uint
}

func NewPPU(bus *bus.Bus) *PPU {
	return &PPU{
		bus:  bus,
		mode: 2,
	}
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

func (ppu *PPU) drawLine() error {
	scanline := [LCD_WIDTH]uint8{}
	y := int(ppu.ly + ppu.scy) // NOTE: wrap around
	tile_y := y / 8
	pix_y := y % 8
	for ax := 0; ax < LCD_WIDTH; ax++ {
		x := int(uint8(ax) + ppu.scx) // NOTE: wrap around
		tile_x := x / 8
		pix_x := x % 8
		tileNo := ppu.Get8(uint16(0x9800 + 32*tile_y + tile_x))
		off := uint16(0x8000 + int(tileNo)*16 + 2*pix_y)

		paletteIdxLSB := (ppu.Get8(off) >> (7 - pix_x)) & 1
		paletteIdxMSB := (ppu.Get8(off+1) >> (7 - pix_x)) & 1
		paletteIdx := paletteIdxLSB | (paletteIdxMSB << 1)
		color := (ppu.BGP() >> (2 * paletteIdx)) & 3
		scanline[x] = color
	}
	ppu.bus.LCD.DrawLine(int(ppu.ly), scanline[:])
	return nil
}

func (ppu *PPU) Update(elapsedTick uint) error {
	ppu.tick += elapsedTick
	switch {
	case ppu.mode == 2 && ppu.tick >= 80: // OAM Search
		ppu.tick -= 80
		ppu.mode = 3

	case ppu.mode == 3 && ppu.tick >= 168: // Pixel Transfer
		ppu.tick -= 168
		ppu.mode = 0
		ppu.drawLine()

	case ppu.mode == 0 && ppu.tick >= 208: // H-Blank
		ppu.tick -= 208
		ppu.ly += 1
		if ppu.ly >= 144 {
			ppu.mode = 1
		} else {
			ppu.mode = 2
		}

	case ppu.mode == 1 && ppu.tick >= 4560: // V-Blank
		ppu.tick -= 4560
		ppu.ly = 0
		ppu.mode = 2
	}

	return nil
}
