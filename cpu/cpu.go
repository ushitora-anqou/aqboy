package cpu

import (
	"fmt"
	"log"
	"math/bits"

	"github.com/ushitora-anqou/aqboy/bus"
	"github.com/ushitora-anqou/aqboy/util"
)

func reg2str(index uint8) string {
	return []string{"B", "C", "D", "E", "H", "L", "(HL)", "A"}[index]
}

func regBC_DE_HL_SP_ToStr(index uint8) string {
	return []string{"BC", "DE", "HL", "SP"}[index]
}

func regBC_DE_HLPLUS_HLMINUS_ToStr(index uint8) string {
	return []string{"BC", "DE", "HL+", "HL-"}[index]
}

func regBC_DE_HL_AF_ToStr(index uint8) string {
	return []string{"BC", "DE", "HL", "AF"}[index]
}

func cc2str(index uint8, needTailComma bool) string {
	if needTailComma {
		return []string{"", "NZ, ", "Z, ", "NC, ", "C, "}[index]
	} else {
		return []string{"", "NZ", "Z", "NC", "C"}[index]
	}
}

func b2u8(b bool) uint8 {
	return util.BoolToU8(b)
}

func rotateRight8(x uint8, k int) uint8 {
	// Thanks to: https://cs.opensource.google/go/go/+/refs/tags/go1.17.6:src/math/bits/bits.go;l=181
	const n = 8
	s := uint(k) & (n - 1)
	return (x >> s) | (x << (n - s))
}

func addN(x, y uint64, carry bool, max uint64) (uint64, bool) {
	sum := x&max + y&max
	if carry {
		sum += 1
	}
	return sum & max, sum > max
}

func add16(x, y uint16, carry bool) (uint16, bool) {
	sum, carry := addN(uint64(x), uint64(y), carry, 0xffff)
	return uint16(sum), carry
}

func add12(x, y uint16, carry bool) (uint16, bool) {
	sum, carry := addN(uint64(x), uint64(y), carry, 0x0fff)
	return uint16(sum), carry
}

func add8(x, y uint8, carry bool) (uint8, bool) {
	sum, carry := addN(uint64(x), uint64(y), carry, 0xff)
	return uint8(sum), carry
}

func add4(x, y uint8, carry bool) (uint8, bool) {
	sum, carry := addN(uint64(x), uint64(y), carry, 0x0f)
	return uint8(sum), carry
}

func sub8(x, y uint8, borrow bool) (uint8, bool) {
	// Thanks to: https://cs.opensource.google/go/go/+/refs/tags/go1.17.6:src/math/bits/bits.go;l=380
	diff := x - y - b2u8(borrow)
	borrowOut := (((^x & y) | (^(x ^ y) & diff)) >> 7) != 0
	return diff, borrowOut
}

func sub4(xu8, yu8 uint8, borrow bool) (uint8, bool) {
	// Thanks to: https://cs.opensource.google/go/go/+/refs/tags/go1.17.6:src/math/bits/bits.go;l=380
	x, y := xu8&0x0f, yu8&0x0f
	diff := (x - y - b2u8(borrow)) & 0x0f
	borrowOut := (((^x & y) | (^(x ^ y) & diff)) >> 3) != 0
	return diff, borrowOut
}

type InterruptBits uint8

func (ib *InterruptBits) getN(i int) bool {
	return ((*ib >> i) & 1) != 0
}

func (ib *InterruptBits) setN(i int, val bool) {
	if val {
		*ib |= (1 << i)
	} else {
		*ib &^= (1 << i)
	}
}

func (ib *InterruptBits) get() uint8 {
	return uint8(*ib)
}

func (ib *InterruptBits) set(val uint8) {
	*ib = InterruptBits(val)
}

type CPU struct {
	bus                    *bus.Bus
	pc, sp                 uint16
	a, f, b, c, d, e, h, l uint8
	ime                    bool // Interrupt Master Enable flag (IME)
	halted                 bool
	intEnable, intFlag     InterruptBits
}

func NewCPU(bus *bus.Bus) *CPU {
	return &CPU{
		bus: bus,
		a:   0x11,
		f:   0x80,
		b:   0x00,
		c:   0x00,
		d:   0xff,
		e:   0x56,
		sp:  0xfffe,
		pc:  0x0100,
		ime: false,
	}
}

func (cpu *CPU) PC() uint16 {
	return cpu.pc
}
func (cpu *CPU) SP() uint16 {
	return cpu.sp
}
func (cpu *CPU) A() uint8 {
	return cpu.a
}
func (cpu *CPU) F() uint8 {
	return cpu.f
}
func (cpu *CPU) B() uint8 {
	return cpu.b
}
func (cpu *CPU) C() uint8 {
	return cpu.c
}
func (cpu *CPU) D() uint8 {
	return cpu.d
}
func (cpu *CPU) E() uint8 {
	return cpu.e
}
func (cpu *CPU) H() uint8 {
	return cpu.h
}
func (cpu *CPU) L() uint8 {
	return cpu.l
}
func (cpu *CPU) AF() uint16 {
	return ((uint16)(cpu.A()) << 8) + (uint16)(cpu.F())
}
func (cpu *CPU) BC() uint16 {
	return ((uint16)(cpu.B()) << 8) + (uint16)(cpu.C())
}
func (cpu *CPU) DE() uint16 {
	return ((uint16)(cpu.D()) << 8) + (uint16)(cpu.E())
}
func (cpu *CPU) HL() uint16 {
	return ((uint16)(cpu.H()) << 8) + (uint16)(cpu.L())
}
func (cpu *CPU) SetPC(pc uint16) {
	cpu.pc = pc
}
func (cpu *CPU) SetSP(sp uint16) {
	cpu.sp = sp
}
func (cpu *CPU) IncPC(val int) {
	cpu.SetPC((uint16)((int)(cpu.PC()) + val))
}
func (cpu *CPU) SetA(a uint8) {
	cpu.a = a
}
func (cpu *CPU) SetF(f uint8) {
	cpu.f = f & 0xf0
}
func (cpu *CPU) SetB(b uint8) {
	cpu.b = b
}
func (cpu *CPU) SetC(c uint8) {
	cpu.c = c
}
func (cpu *CPU) SetD(d uint8) {
	cpu.d = d
}
func (cpu *CPU) SetE(e uint8) {
	cpu.e = e
}
func (cpu *CPU) SetH(h uint8) {
	cpu.h = h
}
func (cpu *CPU) SetL(l uint8) {
	cpu.l = l
}
func (cpu *CPU) SetAF(af uint16) {
	cpu.SetA(uint8(af >> 8))
	cpu.SetF(uint8(af))
}
func (cpu *CPU) SetBC(bc uint16) {
	cpu.SetB(uint8(bc >> 8))
	cpu.SetC(uint8(bc))
}
func (cpu *CPU) SetDE(de uint16) {
	cpu.SetD(uint8(de >> 8))
	cpu.SetE(uint8(de))
}
func (cpu *CPU) SetHL(hl uint16) {
	cpu.SetH(uint8(hl >> 8))
	cpu.SetL(uint8(hl))
}
func (cpu *CPU) IncHL() {
	cpu.SetHL(cpu.HL() + 1)
}
func (cpu *CPU) DecHL() {
	cpu.SetHL(cpu.HL() - 1)
}
func (cpu *CPU) FlagZ() bool {
	return ((cpu.F() & (1 << 7)) != 0)
}
func (cpu *CPU) FlagN() bool {
	return ((cpu.F() & (1 << 6)) != 0)
}
func (cpu *CPU) FlagH() bool {
	return ((cpu.F() & (1 << 5)) != 0)
}
func (cpu *CPU) FlagC() bool {
	return ((cpu.F() & (1 << 4)) != 0)
}
func (cpu *CPU) SetFlag(flag bool, n uint) {
	if flag {
		cpu.SetF(cpu.F() | (1 << n))
	} else {
		cpu.SetF(cpu.F() &^ (1 << n))
	}
}
func (cpu *CPU) SetFlagZ(flag bool) {
	cpu.SetFlag(flag, 7)
}
func (cpu *CPU) SetFlagN(flag bool) {
	cpu.SetFlag(flag, 6)
}
func (cpu *CPU) SetFlagH(flag bool) {
	cpu.SetFlag(flag, 5)
}
func (cpu *CPU) SetFlagC(flag bool) {
	cpu.SetFlag(flag, 4)
}
func (cpu *CPU) SetFlagZNHC(z, n, h, c bool) {
	cpu.SetFlagZ(z)
	cpu.SetFlagN(n)
	cpu.SetFlagH(h)
	cpu.SetFlagC(c)
}
func (cpu *CPU) SetIME(flag bool) {
	cpu.ime = flag
}
func (cpu *CPU) SetIE(val uint8) {
	cpu.intEnable.set(val)
}
func (cpu *CPU) SetIF(val uint8) {
	cpu.intFlag.set(val)
}
func (cpu *CPU) SetHalted(b bool) {
	cpu.halted = b
}
func (cpu *CPU) IME() bool {
	return cpu.ime
}
func (cpu *CPU) IE() uint8 {
	return cpu.intEnable.get()
}
func (cpu *CPU) IF() uint8 {
	return cpu.intFlag.get()
}
func (cpu *CPU) Halted() bool {
	return cpu.halted
}

func (cpu *CPU) getReg(num uint8) uint8 {
	switch num {
	case 0:
		return cpu.B()
	case 1:
		return cpu.C()
	case 2:
		return cpu.D()
	case 3:
		return cpu.E()
	case 4:
		return cpu.H()
	case 5:
		return cpu.L()
	case 6:
		return cpu.bus.MMU.Get8(cpu.HL())
	case 7:
		return cpu.A()
	}
	log.Fatalf("Invalid num: %d", num)
	return 0
}

func (cpu *CPU) getReg16(dst uint8, is3rdSP bool) uint16 {
	switch dst {
	case 0:
		return cpu.BC()
	case 1:
		return cpu.DE()
	case 2:
		return cpu.HL()
	case 3:
		if is3rdSP {
			return cpu.SP()
		} else {
			return cpu.AF()
		}
	default:
		log.Fatalf("Invalid dst: %d", dst)
	}
	return 0
}

func (cpu *CPU) setReg16(dst uint8, val uint16, is3rdSP bool) {
	switch dst {
	case 0:
		cpu.SetBC(val)
	case 1:
		cpu.SetDE(val)
	case 2:
		cpu.SetHL(val)
	case 3:
		if is3rdSP {
			cpu.SetSP(val)
		} else {
			cpu.SetAF(val)
		}
	default:
		log.Fatalf("Invalid dst: %d", dst)
	}
}

func (cpu *CPU) setReg(dst, val uint8) {
	switch dst {
	case 0:
		cpu.SetB(val)
	case 1:
		cpu.SetC(val)
	case 2:
		cpu.SetD(val)
	case 3:
		cpu.SetE(val)
	case 4:
		cpu.SetH(val)
	case 5:
		cpu.SetL(val)
	case 6:
		cpu.bus.MMU.Set8(cpu.HL(), val)
	case 7:
		cpu.SetA(val)
	default:
		log.Fatalf("Invalid num: %d", dst)
	}
	return
}

func (cpu *CPU) incReg(reg uint8) (uint8, bool) {
	src := cpu.getReg(reg)
	res := src + 1
	cpu.setReg(reg, res)
	_, halfCarry := add4(src, 1, false)
	return res, halfCarry
}

func (cpu *CPU) decReg(reg uint8) (uint8, bool) {
	src := cpu.getReg(reg)
	res := src - 1
	cpu.setReg(reg, res)
	_, halfCarry := sub4(src, 1, false)
	return res, halfCarry
}

func (cpu *CPU) addA(rhs uint8) {
	res, c := add8(cpu.A(), rhs, false)
	_, h := add4(cpu.A(), rhs, false)
	cpu.SetA(res)
	cpu.SetFlagZNHC(res == 0, false, h, c)
}

func (cpu *CPU) adcA(rhs uint8) {
	res, c := add8(cpu.A(), rhs, cpu.FlagC())
	_, h := add4(cpu.A(), rhs, cpu.FlagC())
	cpu.SetA(res)
	cpu.SetFlagZNHC(res == 0, false, h, c)
}

func (cpu *CPU) subA(rhs uint8) {
	res, c := sub8(cpu.A(), rhs, false)
	_, h := sub4(cpu.A(), rhs, false)
	cpu.SetA(res)
	cpu.SetFlagZNHC(res == 0, true, h, c)
}

func (cpu *CPU) sbcA(rhs uint8) {
	res, c := sub8(cpu.A(), rhs, cpu.FlagC())
	_, h := sub4(cpu.A(), rhs, cpu.FlagC())
	cpu.SetA(res)
	cpu.SetFlagZNHC(res == 0, true, h, c)
}

func (cpu *CPU) andA(rhs uint8) {
	res := cpu.A() & rhs
	cpu.SetA(res)
	cpu.SetFlagZNHC(res == 0, false, true, false)
}

func (cpu *CPU) xorA(rhs uint8) {
	res := cpu.A() ^ rhs
	cpu.SetA(res)
	cpu.SetFlagZNHC(res == 0, false, false, false)
}

func (cpu *CPU) orA(rhs uint8) {
	res := cpu.A() | rhs
	cpu.SetA(res)
	cpu.SetFlagZNHC(res == 0, false, false, false)
}

func (cpu *CPU) cpA(rhs uint8) {
	src := cpu.A()
	cpu.subA(rhs)
	cpu.SetA(src) // restore
}

func (cpu *CPU) rlc(val uint8) (res uint8, carry bool) {
	res = bits.RotateLeft8(val, 1)
	carry = ((val >> 7) & 1) != 0
	return
}

func (cpu *CPU) rrc(val uint8) (res uint8, carry bool) {
	res = rotateRight8(val, 1)
	carry = ((val >> 0) & 1) != 0
	return
}

func (cpu *CPU) rl(val uint8) (res uint8, carry bool) {
	res = val<<1 | b2u8(cpu.FlagC())
	carry = ((val >> 7) & 1) != 0
	return
}

func (cpu *CPU) rr(val uint8) (res uint8, carry bool) {
	res = b2u8(cpu.FlagC())<<7 | val>>1
	carry = ((val >> 0) & 1) != 0
	return
}

func (cpu *CPU) sla(val uint8) (res uint8, carry bool) {
	res = val << 1
	carry = ((val >> 7) & 1) != 0
	return
}

func (cpu *CPU) sra(val uint8) (res uint8, carry bool) {
	res = val>>1 | val&0x80 /* sign extension */
	carry = ((val >> 0) & 1) != 0
	return
}

func (cpu *CPU) srl(val uint8) (res uint8, carry bool) {
	res = val >> 1
	carry = ((val >> 0) & 1) != 0
	return
}

func (cpu *CPU) push16(val uint16) {
	sp := cpu.SP()
	sp -= 2
	cpu.bus.MMU.Set16(sp, val)
	cpu.SetSP(sp)
}

func (cpu *CPU) pop16() uint16 {
	sp := cpu.SP()
	val := cpu.bus.MMU.Get16(sp)
	sp += 2
	cpu.SetSP(sp)
	return val
}

func (cpu *CPU) call(addr uint16) {
	cpu.push16(cpu.PC())
	cpu.SetPC(addr)
}

func (cpu *CPU) ret() {
	addr := cpu.pop16()
	cpu.SetPC(addr)
}

func (cpu *CPU) addSP8(val uint8) uint16 {
	sp := cpu.SP()
	_, carry := add8(uint8(sp), val, false)
	_, halfCarry := add4(uint8(sp), val, false)
	cpu.SetFlagZNHC(false, false, halfCarry, carry)
	return sp + uint16(int8(val)) // NOTE: sign extension
}

func (cpu *CPU) handleInterrupt() uint {
	var tick uint
	for i := 0; i < 5; i++ {
		if !(cpu.intEnable.getN(i) && cpu.intFlag.getN(i)) {
			continue
		}
		if !cpu.IME() && cpu.Halted() {
			cpu.SetHalted(false)
		}
		if cpu.IME() {
			cpu.push16(cpu.PC())
			cpu.SetPC(uint16(0x40 + 0x08*i))
			cpu.intFlag.setN(i, false)
			cpu.SetIME(false)
			cpu.SetHalted(false)
			tick = 12 // FIXME: Correct?
		}
		break
	}
	return tick
}

func getOpTick(opcode, opcode2 uint8, taken bool) uint {
	switch {
	case opcode == 0x20 || opcode == 0x28 || opcode == 0x30 || opcode == 0x38: // JR
		if taken {
			return 12
		} else {
			return 8
		}

	case opcode == 0xc0 || opcode == 0xc8 || opcode == 0xd0 || opcode == 0xd8: // RET
		if taken {
			return 20
		} else {
			return 8
		}

	case opcode == 0xc2 || opcode == 0xca || opcode == 0xd2 || opcode == 0xda: // JP
		if taken {
			return 16
		} else {
			return 12
		}

	case opcode == 0xc4 || opcode == 0xcc || opcode == 0xd4 || opcode == 0xdc: // CALL
		if taken {
			return 24
		} else {
			return 12
		}

	case opcode == 0xcb: // PREFIX CB
		return []uint{
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
			8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8,
			8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8,
			8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8,
			8, 8, 8, 8, 8, 8, 12, 8, 8, 8, 8, 8, 8, 8, 12, 8,
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
			8, 8, 8, 8, 8, 8, 16, 8, 8, 8, 8, 8, 8, 8, 16, 8,
		}[opcode2]
	}

	return []uint{
		4, 12, 8, 8, 4, 4, 8, 4, 20, 8, 8, 8, 4, 4, 8, 4, // 0x
		4, 12, 8, 8, 4, 4, 8, 4, 12, 8, 8, 8, 4, 4, 8, 4, // 1x
		0, 12, 8, 8, 4, 4, 8, 4, 0, 8, 8, 8, 4, 4, 8, 4, // 2x
		0, 12, 8, 8, 12, 12, 12, 4, 0, 8, 8, 8, 4, 4, 8, 4, // 3x
		4, 4, 4, 4, 4, 4, 8, 4, 4, 4, 4, 4, 4, 4, 8, 4, // 4x
		4, 4, 4, 4, 4, 4, 8, 4, 4, 4, 4, 4, 4, 4, 8, 4, // 5x
		4, 4, 4, 4, 4, 4, 8, 4, 4, 4, 4, 4, 4, 4, 8, 4, // 6x
		8, 8, 8, 8, 8, 8, 4, 8, 4, 4, 4, 4, 4, 4, 8, 4, // 7x
		4, 4, 4, 4, 4, 4, 8, 4, 4, 4, 4, 4, 4, 4, 8, 4, // 8x
		4, 4, 4, 4, 4, 4, 8, 4, 4, 4, 4, 4, 4, 4, 8, 4, // 9x
		4, 4, 4, 4, 4, 4, 8, 4, 4, 4, 4, 4, 4, 4, 8, 4, // ax
		4, 4, 4, 4, 4, 4, 8, 4, 4, 4, 4, 4, 4, 4, 8, 4, // bx
		0, 12, 0, 16, 0, 16, 8, 16, 0, 16, 0, 4, 0, 24, 8, 16, // cx
		0, 12, 0, 0, 0, 16, 8, 16, 0, 16, 0, 0, 0, 0, 8, 16, // dx
		12, 12, 8, 0, 0, 16, 8, 16, 16, 4, 16, 0, 0, 0, 8, 16, // ex
		12, 12, 8, 4, 0, 16, 8, 16, 12, 8, 16, 4, 0, 0, 8, 16, // fx
	}[opcode]
}

func (cpu *CPU) stepCB() {
	opcode := cpu.bus.MMU.Get8(cpu.PC())
	reg := opcode % 8
	regVal := cpu.getReg(reg)
	res := regVal
	z, n, h, c := cpu.FlagZ(), cpu.FlagN(), cpu.FlagH(), cpu.FlagC()

	switch {
	case 0x00 <= opcode && opcode <= 0x07: // RLC (B|C|D|E|H|L|(HL)|A)
		cpu.traceInst1("RLC %s", reg2str(reg))
		res, c = cpu.rlc(regVal)
		z, n, h = res == 0, false, false

	case 0x08 <= opcode && opcode <= 0x0f: // RRC (B|C|D|E|H|L|(HL)|A)
		cpu.traceInst1("RRC %s", reg2str(reg))
		res, c = cpu.rrc(regVal)
		z, n, h = res == 0, false, false

	case 0x10 <= opcode && opcode <= 0x17: // RL (B|C|D|E|H|L|(HL)|A)
		cpu.traceInst1("RL %s", reg2str(reg))
		res, c = cpu.rl(regVal)
		z, n, h = res == 0, false, false

	case 0x18 <= opcode && opcode <= 0x1f: // RR (B|C|D|E|H|L|(HL)|A)
		cpu.traceInst1("RR %s", reg2str(reg))
		res, c = cpu.rr(regVal)
		z, n, h = res == 0, false, false

	case 0x20 <= opcode && opcode <= 0x27: // SLA (B|C|D|E|H|L|(HL)|A)
		cpu.traceInst1("SLA %s", reg2str(reg))
		res, c = cpu.sla(regVal)
		z, n, h = res == 0, false, false

	case 0x28 <= opcode && opcode <= 0x2f: // SRA (B|C|D|E|H|L|(HL)|A)
		cpu.traceInst1("SRA %s", reg2str(reg))
		res, c = cpu.sra(regVal)
		z, n, h = res == 0, false, false

	case 0x30 <= opcode && opcode <= 0x37: // SWAP (B|C|D|E|H|L|(HL)|A)
		cpu.traceInst1("SWAP %s", reg2str(reg))
		res = (regVal >> 4) | (regVal << 4)
		z, n, h, c = res == 0, false, false, false

	case 0x38 <= opcode && opcode <= 0x3f: // SRL (B|C|D|E|H|L|(HL)|A)
		cpu.traceInst1("SRL %s", reg2str(reg))
		res, c = cpu.srl(regVal)
		z, n, h = res == 0, false, false

	case 0x40 <= opcode && opcode <= 0x7f: // BIT 0-7, (B|C|D|E|H|L|(HL)|A)
		index := (opcode - 0x40) / 8
		cpu.traceInst2("BIT %d, %s", index, reg2str(reg))
		z, n, h, c = ((regVal>>int(index))&1) == 0, false, true, cpu.FlagC()

	case 0x80 <= opcode && opcode <= 0xbf: // RES 0-7, (B|C|D|E|H|L|(HL)|A)
		index := (opcode - 0x80) / 8
		cpu.traceInst2("RES %d, %s", index, reg2str(reg))
		res = regVal &^ (1 << index)

	case 0xc0 <= opcode && opcode <= 0xff: // SET 0-7, (B|C|D|E|H|L|(HL)|A)
		index := (opcode - 0xc0) / 8
		cpu.traceInst2("SET %d, %s", index, reg2str(reg))
		res = regVal | (1 << index)
	}
	cpu.setReg(reg, res)
	cpu.SetFlagZNHC(z, n, h, c)
	cpu.IncPC(1)
}

func (cpu *CPU) Step() (uint, error) {
	interruptTick := cpu.handleInterrupt()

	if cpu.Halted() {
		if util.IsTraceEnabled() {
			// FIXME: This guard is semantically not necessary.
			// However, without this guard, the performance of this emulator becomes
			// very bad. I need to find the reason for this.
			cpu.traceInst0("(halted)")
		}
		return 4, nil
	}

	mmu := cpu.bus.MMU
	opcode := mmu.Get8(cpu.PC())
	opLow := opcode & 0x0f
	opHigh := opcode >> 4
	imm8 := mmu.Get8(cpu.PC() + 1)
	imm16 := mmu.Get16(cpu.PC() + 1)
	taken := false

	switch {

	case opcode == 0x00: // NOP
		cpu.traceInst0("NOP")
		cpu.IncPC(1)

	case opLow == 0x01 && (0 <= opHigh && opHigh <= 3): // LD (BC|DE|HL|SP), d16
		cpu.traceInst2("LD %s, 0x%x", regBC_DE_HL_SP_ToStr(opHigh), imm16)
		cpu.setReg16(opHigh, imm16, true)
		cpu.IncPC(3)

	case opLow == 0x02 && (0 <= opHigh && opHigh <= 3): // LD ((BC)|(DE)|(HL+)|(HL-)), A
		cpu.traceInst1("LD (%s), A", regBC_DE_HLPLUS_HLMINUS_ToStr(opHigh))
		switch opHigh {
		case 0:
			mmu.Set8(cpu.BC(), cpu.A())
		case 1:
			mmu.Set8(cpu.DE(), cpu.A())
		case 2:
			mmu.Set8(cpu.HL(), cpu.A())
			cpu.IncHL()
		case 3:
			mmu.Set8(cpu.HL(), cpu.A())
			cpu.DecHL()
		}
		cpu.IncPC(1)

	case (opLow == 0x3) && (0 <= opHigh && opHigh <= 3): // INC (BC|DE|HL|SP)
		index := opHigh
		cpu.traceInst1("INC %s", regBC_DE_HL_SP_ToStr(index))
		val := cpu.getReg16(index, true)
		cpu.setReg16(index, val+1, true)
		cpu.IncPC(1)

	case (opLow%8 == 4) && (0 <= opHigh && opHigh <= 3): // INC (B|C|D|E|H|L|(HL)|A)
		reg := (opcode - 0x04) / 8
		cpu.traceInst1("INC %s", reg2str(reg))
		val, halfCarry := cpu.incReg(reg)
		cpu.SetFlagZNHC(val == 0, false, halfCarry, cpu.FlagC())
		cpu.IncPC(1)

	case (opLow%8 == 5) && (0 <= opHigh && opHigh <= 3): // DEC (B|C|D|E|H|L|(HL)|A)
		reg := (opcode - 0x05) / 8
		cpu.traceInst1("DEC %s", reg2str(reg))
		val, halfCarry := cpu.decReg(reg)
		cpu.SetFlagZNHC(val == 0, true, halfCarry, cpu.FlagC())
		cpu.IncPC(1)

	case (opLow%8 == 6) && (0 <= opHigh && opHigh <= 3): // LD (B|C|D|E|H|L|(HL)|A), d8
		reg := (opcode - 0x06) / 8
		cpu.traceInst2("LD %s, 0x%x", reg2str(reg), imm8)
		cpu.setReg(reg, imm8)
		cpu.IncPC(2)

	case (opLow%8 == 7) && (0 <= opHigh && opHigh <= 1): // RLCA|RRCA|RLA|RRA
		var res uint8
		var carry bool
		switch opcode {
		case 0x07: // RLCA
			cpu.traceInst0("RLCA")
			res, carry = cpu.rlc(cpu.A())
		case 0x0f: // RRCA
			cpu.traceInst0("RRCA")
			res, carry = cpu.rrc(cpu.A())
		case 0x17: // RLA
			cpu.traceInst0("RLA")
			res, carry = cpu.rl(cpu.A())
		case 0x1f: // RRA
			cpu.traceInst0("RRA")
			res, carry = cpu.rr(cpu.A())
		}
		cpu.SetA(res)
		cpu.SetFlagZNHC(false, false, false, carry)
		cpu.IncPC(1)

	case opcode == 0x08: // LD (a16), SP
		cpu.traceInst1("LD (0x%04x), SP", imm16)
		mmu.Set16(imm16, cpu.SP())
		cpu.IncPC(3)

	case (opLow == 0x9) && (0 <= opHigh && opHigh <= 3): // ADD HL, (BC|DE|HL|SP)
		index := opHigh
		cpu.traceInst1("ADD HL, %s", regBC_DE_HL_SP_ToStr(index))
		rhs := cpu.getReg16(index, true)
		res, carry := add16(cpu.HL(), rhs, false)
		_, halfCarry := add12(cpu.HL(), rhs, false)
		cpu.SetHL(res)
		cpu.SetFlagZNHC(cpu.FlagZ(), false, halfCarry, carry)
		cpu.IncPC(1)

	case opLow == 0xa && (0 <= opHigh && opHigh <= 3): // LD A, ((BC)|(DE)|(HL+)|(HL-))
		cpu.traceInst1("LD A, (%s)", regBC_DE_HLPLUS_HLMINUS_ToStr(opHigh))
		switch opHigh {
		case 0:
			cpu.SetA(mmu.Get8(cpu.BC()))
		case 1:
			cpu.SetA(mmu.Get8(cpu.DE()))
		case 2:
			cpu.SetA(mmu.Get8(cpu.HL()))
			cpu.IncHL()
		case 3:
			cpu.SetA(mmu.Get8(cpu.HL()))
			cpu.DecHL()
		}
		cpu.IncPC(1)

	case opcode == 0x10: // STOP
		cpu.traceInst0("STOP")
		cpu.IncPC(2)

	case (opLow == 0xb) && (0 <= opHigh && opHigh <= 3): // DEC (BC|DE|HL|SP)
		index := opHigh
		cpu.traceInst1("DEC %s", regBC_DE_HL_SP_ToStr(index))
		val := cpu.getReg16(index, true)
		cpu.setReg16(index, val-1, true)
		cpu.IncPC(1)

	case opcode == 0x18 || // JR r8
		((opLow == 0 || opLow == 8) && (opHigh == 2 || opHigh == 3)): // JR (NZ|Z|NC|C), r8
		cpu.traceInst2("JR %s0x%x", cc2str((opcode-0x18)/8, true), imm8)
		if opcode == 0x18 ||
			(opcode == 0x20 && !cpu.FlagZ()) || (opcode == 0x28 && cpu.FlagZ()) ||
			(opcode == 0x30 && !cpu.FlagC()) || (opcode == 0x38 && cpu.FlagC()) {
			cpu.IncPC(int(int8(imm8)))
			taken = true
		}
		cpu.IncPC(2)

	case opcode == 0x27: // DAA
		cpu.traceInst0("DAA")

		// Thanks to: https://forums.nesdev.org/viewtopic.php?t=15944
		a := cpu.A()
		n, h, c := cpu.FlagN(), cpu.FlagH(), cpu.FlagC()
		if n { // After a subtraction, only adjust if (half-)carry occurred
			if c {
				a -= 0x60
			}
			if h {
				a -= 0x06
			}
		} else { // After an addition, adjust if (half-)carry occurred or if result is out of bounds
			if c || a > 0x99 {
				a += 0x60
				c = true
			}
			if h || (a&0x0f) > 0x09 {
				a += 0x06
			}
		}
		cpu.SetA(a)
		cpu.SetFlagZNHC(a == 0, n, false, c)
		cpu.IncPC(1)

	case opcode == 0x2f: // CPL
		cpu.traceInst0("CPL")
		cpu.SetA(^cpu.A())
		cpu.SetFlagZNHC(cpu.FlagZ(), true, true, cpu.FlagC())
		cpu.IncPC(1)

	case opcode == 0x37: // SCF
		cpu.traceInst0("SCF")
		cpu.SetFlagZNHC(cpu.FlagZ(), false, false, true)
		cpu.IncPC(1)

	case opcode == 0x3f: // CCF
		cpu.traceInst0("CCF")
		cpu.SetFlagZNHC(cpu.FlagZ(), false, false, !cpu.FlagC())
		cpu.IncPC(1)

	case 0x40 <= opcode && opcode <= 0x7f && opcode != 0x76 /* not HALT */ : // LD reg1,reg2
		reg1 := (opcode & 0x3f) >> 3
		reg2 := (opcode & 0x07)
		cpu.traceInst2("LD %s, %s", reg2str(reg1), reg2str(reg2))
		val := cpu.getReg(reg2)
		cpu.setReg(reg1, val)
		cpu.IncPC(1)

	case opcode == 0x76: // HALT
		cpu.traceInst0("HALT")
		cpu.SetHalted(true)
		cpu.IncPC(1)

	case 0x80 <= opcode && opcode <= 0xbf:
		reg := (opcode & 0x07)
		val := cpu.getReg(reg)
		switch {
		case 0x80 <= opcode && opcode <= 0x87: // ADD A, reg
			cpu.traceInst1("ADD A, %s", reg2str(reg))
			cpu.addA(val)
		case 0x88 <= opcode && opcode <= 0x8f: // ADC A, reg
			cpu.traceInst1("ADC A, %s", reg2str(reg))
			cpu.adcA(val)
		case 0x90 <= opcode && opcode <= 0x97: // SUB reg
			cpu.traceInst1("SUB %s", reg2str(reg))
			cpu.subA(val)
		case 0x98 <= opcode && opcode <= 0x9f: // SBC A, reg
			cpu.traceInst1("SBC A, %s", reg2str(reg))
			cpu.sbcA(val)
		case 0xa0 <= opcode && opcode <= 0xa7: // AND reg
			cpu.traceInst1("AND %s", reg2str(reg))
			cpu.andA(val)
		case 0xa8 <= opcode && opcode <= 0xaf: // XOR reg
			cpu.traceInst1("XOR %s", reg2str(reg))
			cpu.xorA(val)
		case 0xb0 <= opcode && opcode <= 0xb7: // OR reg
			cpu.traceInst1("OR %s", reg2str(reg))
			cpu.orA(val)
		case 0xb8 <= opcode && opcode <= 0xbf: // CP reg
			cpu.traceInst1("CP %s", reg2str(reg))
			cpu.cpA(val)
		}
		cpu.IncPC(1)

	case opcode == 0xc9 || // RET
		(opLow%8 == 0 && (opHigh == 0xc || opHigh == 0xd)): // RET (NZ|Z|NC|C)
		var strIdx uint8 = 0
		if opLow%8 == 0 {
			strIdx = (opcode - 0xc0) / 8
		}
		cpu.traceInst1("RET %s", cc2str(strIdx, false))
		if opcode == 0xc9 ||
			(opcode == 0xc0 && !cpu.FlagZ()) || (opcode == 0xc8 && cpu.FlagZ()) ||
			(opcode == 0xd0 && !cpu.FlagC()) || (opcode == 0xd8 && cpu.FlagC()) {
			cpu.ret()
			taken = true
		} else {
			cpu.IncPC(1)
		}

	case opLow == 1 && (0xc <= opHigh && opHigh <= 0xf): // POP
		index := opHigh - 0xc
		cpu.traceInst1("POP %s", regBC_DE_HL_AF_ToStr(index))
		cpu.setReg16(index, cpu.pop16(), false)
		cpu.IncPC(1)

	case opcode == 0xc3 || // JP a16
		(opLow%8 == 2 && (opHigh == 0xc || opHigh == 0xd)): // JP (NZ|Z|NC|C), a16
		var strIdx uint8 = 0
		if opLow%8 == 2 {
			strIdx = (opcode - 0xc2) / 8
		}
		cpu.traceInst2("JP %s0x%x", cc2str(strIdx, true), imm16)
		if opcode == 0xc3 ||
			(opcode == 0xc2 && !cpu.FlagZ()) || (opcode == 0xca && cpu.FlagZ()) ||
			(opcode == 0xd2 && !cpu.FlagC()) || (opcode == 0xda && cpu.FlagC()) {
			cpu.SetPC(imm16)
			taken = true
		} else {
			cpu.IncPC(3)
		}

	case opcode == 0xcd || // CALL a16
		(opLow%8 == 4 && (opHigh == 0xc || opHigh == 0xd)): // CALL (NZ|Z|NC|C), a16
		var strIdx uint8 = 0
		if opLow%8 == 4 {
			strIdx = (opcode - 0xc4) / 8
		}
		cpu.traceInst2("CALL %s0x%x", cc2str(strIdx, true), imm16)
		cpu.IncPC(3)
		if opcode == 0xcd ||
			(opcode == 0xc4 && !cpu.FlagZ()) || (opcode == 0xcc && cpu.FlagZ()) ||
			(opcode == 0xd4 && !cpu.FlagC()) || (opcode == 0xdc && cpu.FlagC()) {
			cpu.call(imm16)
			taken = true
		}

	case opLow == 5 && (0xc <= opHigh && opHigh <= 0xf):
		index := opHigh - 0xc
		cpu.traceInst1("PUSH %s", regBC_DE_HL_AF_ToStr(index))
		cpu.push16(cpu.getReg16(index, false))
		cpu.IncPC(1)

	case opLow%8 == 6 && (0xc <= opHigh && opHigh <= 0xf):
		switch opcode {
		case 0xc6: // ADD A, d8
			cpu.traceInst1("ADD A, 0x%x", imm8)
			cpu.addA(imm8)
		case 0xce: // ADC A, d8
			cpu.traceInst1("ADC A, 0x%x", imm8)
			cpu.adcA(imm8)
		case 0xd6: // SUB d8
			cpu.traceInst1("SUB 0x%x", imm8)
			cpu.subA(imm8)
		case 0xde: // SBC d8
			cpu.traceInst1("SBC 0x%x", imm8)
			cpu.sbcA(imm8)
		case 0xe6: // AND d8
			cpu.traceInst1("AND 0x%x", imm8)
			cpu.andA(imm8)
		case 0xee: // XOR d8
			cpu.traceInst1("XOR 0x%x", imm8)
			cpu.xorA(imm8)
		case 0xf6: // OR d8
			cpu.traceInst1("OR 0x%x", imm8)
			cpu.orA(imm8)
		case 0xfe: // CP d8
			cpu.traceInst1("CP 0x%x", imm8)
			cpu.cpA(imm8)
		}
		cpu.IncPC(2)

	case opLow%8 == 7 && (0xc <= opHigh && opHigh <= 0xf):
		index := opcode - 0xc7
		cpu.traceInst1("RST %02xH", index)
		cpu.IncPC(1)
		cpu.call(uint16(index))

	case opcode == 0xcb: // PREFIX CB
		cpu.traceInst0("PREFIX CB")
		cpu.IncPC(1)
		cpu.stepCB()

	case opcode == 0xd9: // RETI
		cpu.traceInst0("RETI")
		cpu.ret()
		cpu.SetIME(true)

	case opcode == 0xe0 || opcode == 0xf0:
		if opcode == 0xe0 {
			cpu.traceInst1("LDH (0x%x), A", imm8)
			addr := 0xff00 + uint16(imm8)
			mmu.Set8(addr, cpu.A())
		} else {
			cpu.traceInst1("LDH A, (0x%x)", imm8)
			addr := 0xff00 + uint16(imm8)
			cpu.SetA(mmu.Get8(addr))
		}
		cpu.IncPC(2)

	case opcode == 0xe2 || opcode == 0xf2:
		addr := 0xff00 | uint16(cpu.C())
		if opcode == 0xe2 {
			cpu.traceInst0("LD (C), A")
			mmu.Set8(addr, cpu.A())
		} else {
			cpu.traceInst0("LD A, (C)")
			cpu.SetA(mmu.Get8(addr))
		}
		cpu.IncPC(1)

	case opcode == 0xe8: // ADD SP, r8
		cpu.traceInst1("ADD SP, 0x%x", imm8)
		res := cpu.addSP8(imm8)
		cpu.SetSP(res)
		cpu.IncPC(2)

	case opcode == 0xe9: // JP (HL)
		cpu.traceInst0("JP (HL)")
		addr := cpu.HL()
		cpu.SetPC(addr)

	case opcode == 0xea || opcode == 0xfa:
		if opcode == 0xea {
			cpu.traceInst1("LD (0x%x), A", imm16)
			mmu.Set8(imm16, cpu.A())
		} else {
			cpu.traceInst1("LD A, (0x%x)", imm16)
			cpu.SetA(mmu.Get8(imm16))
		}
		cpu.IncPC(3)

	case opcode == 0xf3 || opcode == 0xfb:
		if opcode == 0xf3 {
			cpu.traceInst0("DI")
			cpu.SetIME(false) // FIXME: Correct? Probably it should be delayed.
		} else {
			cpu.traceInst0("EI")
			cpu.SetIME(true) // FIXME: Correct? Probably it should be delayed.
		}
		cpu.IncPC(1)

	case opcode == 0xf8: // LD HL, SP+r8
		cpu.traceInst1("LD HL, SP+0x%x", imm8)
		res := cpu.addSP8(imm8)
		cpu.SetHL(res)
		cpu.IncPC(2)

	case opcode == 0xf9: // LD SP, HL
		cpu.traceInst0("LD SP, HL")
		cpu.SetSP(cpu.HL())
		cpu.IncPC(1)

	default:
		return 0, fmt.Errorf("Illegal instr: 0x%02x at 0x%04x\n", mmu.Get8(cpu.PC()), cpu.PC())
	}

	tick := getOpTick(opcode, imm8, taken)

	return tick + interruptTick, nil
}

func (cpu *CPU) traceInst0(format string) {
	util.Trace1("0x%04x: "+format, cpu.PC())
}

/*
	Functions traceInstK written below are generated by the following Ruby script:

		def foo(i)
		  args = i.times.map { |k| "v#{k}" }.join(", ")
		  puts "func (cpu *CPU) traceInst#{i}(format string, #{args} interface{}) {"
		  puts "\tutil.Trace#{i+1}(\"0x%04x: \"+format, cpu.PC(), #{args})"
		  puts "}"
		end
		(1...9).each { |i| foo(i); puts "\n" }
*/
func (cpu *CPU) traceInst1(format string, v0 interface{}) {
	util.Trace2("0x%04x: "+format, cpu.PC(), v0)
}

func (cpu *CPU) traceInst2(format string, v0, v1 interface{}) {
	util.Trace3("0x%04x: "+format, cpu.PC(), v0, v1)
}

func (cpu *CPU) traceInst3(format string, v0, v1, v2 interface{}) {
	util.Trace4("0x%04x: "+format, cpu.PC(), v0, v1, v2)
}

func (cpu *CPU) traceInst4(format string, v0, v1, v2, v3 interface{}) {
	util.Trace5("0x%04x: "+format, cpu.PC(), v0, v1, v2, v3)
}

func (cpu *CPU) traceInst5(format string, v0, v1, v2, v3, v4 interface{}) {
	util.Trace6("0x%04x: "+format, cpu.PC(), v0, v1, v2, v3, v4)
}

func (cpu *CPU) traceInst6(format string, v0, v1, v2, v3, v4, v5 interface{}) {
	util.Trace7("0x%04x: "+format, cpu.PC(), v0, v1, v2, v3, v4, v5)
}

func (cpu *CPU) traceInst7(format string, v0, v1, v2, v3, v4, v5, v6 interface{}) {
	util.Trace8("0x%04x: "+format, cpu.PC(), v0, v1, v2, v3, v4, v5, v6)
}

func (cpu *CPU) traceInst8(format string, v0, v1, v2, v3, v4, v5, v6, v7 interface{}) {
	util.Trace9("0x%04x: "+format, cpu.PC(), v0, v1, v2, v3, v4, v5, v6, v7)
}
