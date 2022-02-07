package ppu

import "github.com/ushitora-anqou/aqboy/bus"

type object struct {
	oamIndex              int
	y, x, tileIndex, attr uint8
}

func newObject(mmu bus.MMU, addr uint16) *object {
	return &object{
		oamIndex:  int(addr-0xfe00) / 4,
		y:         mmu.Get8(addr),
		x:         mmu.Get8(addr + 1),
		tileIndex: mmu.Get8(addr + 2),
		attr:      mmu.Get8(addr + 3),
	}
}

func (o *object) screenY() int {
	return int(o.y) - 16
}

func (o *object) screenX() int {
	return int(o.x) - 8
}

func (o *object) paletteNumber() bool {
	return ((o.attr >> 4) & 1) != 0
}

func (o *object) xFlip() bool {
	return ((o.attr >> 5) & 1) != 0
}

func (o *object) yFlip() bool {
	return ((o.attr >> 6) & 1) != 0
}

func (o *object) overOBJ() bool {
	return ((o.attr >> 7) & 1) != 0
}

type byXAndOAMIndex []*object

func (o byXAndOAMIndex) Len() int {
	return len(o)
}
func (o byXAndOAMIndex) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}
func (o byXAndOAMIndex) Less(i, j int) bool {
	return o[i].x < o[j].x || (o[i].x == o[j].x && o[i].oamIndex < o[j].oamIndex)
}
