package cpu

import (
	"fmt"
	"log"

	"github.com/ushitora-anqou/aqboy/mmu"
)

func dbgpr(format string, v ...interface{}) {
	log.Printf(format, v...)
}

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

func compl(v uint8) uint8 {
	return 0xff ^ v
}

func b2u8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func bitN8(n uint8, index int) bool {
	return ((n >> index) & 1) != 0
}

func add8(x, y uint8, carry bool) (uint8, bool) {
	// Thanks to: https://cs.opensource.google/go/go/+/refs/tags/go1.17.6:src/math/bits/bits.go;l=354
	sum := x + y + b2u8(carry)
	carryOut := (((x & y) | ((x | y) &^ sum)) >> 7) != 0
	return sum, carryOut
}

func add4(xu8, yu8 uint8, carry bool) (uint8, bool) {
	// Thanks to: https://cs.opensource.google/go/go/+/refs/tags/go1.17.6:src/math/bits/bits.go;l=354
	x, y := xu8&0x0f, yu8&0x0f
	sum := (x + y + b2u8(carry)) & 0x0f
	carryOut := (((x & y) | ((x | y) &^ sum)) >> 3) != 0
	return sum, carryOut
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

type CPU struct {
	pc, sp                 uint16
	a, f, b, c, d, e, h, l uint8
	ime                    bool // Interrupt Master Enable flag (IME)
}

func NewCPU() *CPU {
	cpu := &CPU{}
	cpu.a = 0x11
	cpu.f = 0x80
	cpu.b = 0x00
	cpu.c = 0x00
	cpu.d = 0xff
	cpu.e = 0x56
	cpu.sp = 0xfffe
	cpu.pc = 0x100
	cpu.ime = false
	return cpu
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
	return ((uint16)(cpu.a) << 8) + (uint16)(cpu.f)
}
func (cpu *CPU) BC() uint16 {
	return ((uint16)(cpu.b) << 8) + (uint16)(cpu.c)
}
func (cpu *CPU) DE() uint16 {
	return ((uint16)(cpu.d) << 8) + (uint16)(cpu.e)
}
func (cpu *CPU) HL() uint16 {
	return ((uint16)(cpu.h) << 8) + (uint16)(cpu.l)
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
	cpu.f = f
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
	cpu.a = (uint8)(af >> 8)
	cpu.f = (uint8)(af)
}
func (cpu *CPU) SetBC(bc uint16) {
	cpu.b = (uint8)(bc >> 8)
	cpu.c = (uint8)(bc)
}
func (cpu *CPU) SetDE(de uint16) {
	cpu.d = (uint8)(de >> 8)
	cpu.e = (uint8)(de)
}
func (cpu *CPU) SetHL(hl uint16) {
	cpu.h = (uint8)(hl >> 8)
	cpu.l = (uint8)(hl)
}
func (cpu *CPU) IncHL() {
	cpu.SetHL(cpu.HL() + 1)
}
func (cpu *CPU) DecHL() {
	cpu.SetHL(cpu.HL() - 1)
}
func (cpu *CPU) FlagZ() bool {
	return ((cpu.f & (1 << 7)) != 0)
}
func (cpu *CPU) FlagN() bool {
	return ((cpu.f & (1 << 6)) != 0)
}
func (cpu *CPU) FlagH() bool {
	return ((cpu.f & (1 << 5)) != 0)
}
func (cpu *CPU) FlagC() bool {
	return ((cpu.f & (1 << 4)) != 0)
}
func (cpu *CPU) SetFlag(flag bool, n uint) {
	if flag {
		cpu.f |= (1 << n)
	} else {
		cpu.f &= compl(1 << n)
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

func (cpu *CPU) getReg(mmu *mmu.MMU, num uint8) uint8 {
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
		return mmu.Get8(cpu.HL())
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

func (cpu *CPU) setReg16(mmu *mmu.MMU, dst uint8, val uint16, is3rdSP bool) {
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

func (cpu *CPU) setReg(mmu *mmu.MMU, dst, val uint8) {
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
		mmu.Set8(cpu.HL(), val)
	case 7:
		cpu.SetA(val)
	default:
		log.Fatalf("Invalid num: %d", dst)
	}
	return
}

func (cpu *CPU) incReg(mmu *mmu.MMU, reg uint8) (uint8, bool) {
	src := cpu.getReg(mmu, reg)
	res := src + 1
	cpu.setReg(mmu, reg, res)
	_, halfCarry := add4(src, 1, false)
	return res, halfCarry
}

func (cpu *CPU) decReg(mmu *mmu.MMU, reg uint8) (uint8, bool) {
	src := cpu.getReg(mmu, reg)
	res := src - 1
	cpu.setReg(mmu, reg, res)
	_, halfCarry := sub4(src, 1, false)
	return res, halfCarry
}

func (cpu *CPU) push16(mmu *mmu.MMU, val uint16) {
	sp := cpu.SP()
	sp -= 2
	mmu.Set16(sp, val)
	cpu.SetSP(sp)
}

func (cpu *CPU) pop16(mmu *mmu.MMU) uint16 {
	sp := cpu.SP()
	val := mmu.Get16(sp)
	sp += 2
	cpu.SetSP(sp)
	return val
}

func (cpu *CPU) Step(mmu *mmu.MMU) error {
	opcode := mmu.Get8(cpu.PC())
	opLow := opcode & 0x0f
	opHigh := opcode >> 4
	imm8 := mmu.Get8(cpu.PC() + 1)
	imm16 := mmu.Get16(cpu.PC() + 1)

	switch {
	case opcode == 0x00: // NOP
		dbgpr("0x%08x: NOP", cpu.PC())
		cpu.IncPC(1)

	case opcode == 0x76: // HALT
		dbgpr("0x%08x: HALT", cpu.PC())
		return fmt.Errorf("HALT is not supported yet")

	case opcode == 0x18 || // JR r8
		((opLow == 0 || opLow == 8) && (opHigh == 2 || opHigh == 3)): // JR (NZ|Z|NC|C), r8
		dbgpr("0x%08x: JR %s0x%x", cpu.PC(),
			[]string{"", "NZ, ", "Z, ", "NC, ", "C, "}[(opcode-0x18)>>3], imm8)
		if opcode == 0x18 ||
			(opcode == 0x20 && !cpu.FlagZ()) || (opcode == 0x28 && cpu.FlagZ()) ||
			(opcode == 0x30 && !cpu.FlagC()) || (opcode == 0x38 && cpu.FlagC()) {
			cpu.IncPC(int(int8(imm8)))
		}
		cpu.IncPC(2)

	case opLow == 0x01 && (0 <= opHigh && opHigh <= 3): // LD (BC|DE|HL|SP), d16
		dbgpr("0x%08x: LD %s, 0x%x", cpu.PC(), regBC_DE_HL_SP_ToStr(opHigh), imm16)
		cpu.setReg16(mmu, opHigh, imm16, true)
		cpu.IncPC(3)

	case opLow == 0x02 && (0 <= opHigh && opHigh <= 3): // LD ((BC)|(DE)|(HL+)|(HL-)), A
		dbgpr("0x%08x: LD (%s), A", cpu.PC(), regBC_DE_HLPLUS_HLMINUS_ToStr(opHigh))
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

	case opLow == 0x0a && (0 <= opHigh && opHigh <= 3): // LD A, ((BC)|(DE)|(HL+)|(HL-))
		dbgpr("0x%08x: LD A, (%s)", cpu.PC(), regBC_DE_HLPLUS_HLMINUS_ToStr(opHigh))
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

	case (opLow == 0x3) && (0 <= opHigh && opHigh <= 3): // INC (BC|DE|HL|SP)
		index := opHigh
		dbgpr("0x%08x: INC %s", cpu.PC(), regBC_DE_HL_SP_ToStr(index))
		val := cpu.getReg16(index, true)
		cpu.setReg16(mmu, index, val+1, true)
		cpu.IncPC(1)

	case (opLow == 0xb) && (0 <= opHigh && opHigh <= 3): // DEC (BC|DE|HL|SP)
		index := opHigh
		dbgpr("0x%08x: DEC %s", cpu.PC(), regBC_DE_HL_SP_ToStr(index))
		val := cpu.getReg16(index, true)
		cpu.setReg16(mmu, index, val-1, true)
		cpu.IncPC(1)

	case (opLow%8 == 4) && (0 <= opHigh && opHigh <= 3): // INC (B|C|D|E|H|L|(HL)|A)
		reg := opHigh*2 + (opLow-4)/8
		dbgpr("0x%08x: INC %s", cpu.PC(), reg2str(reg))
		val, halfCarry := cpu.incReg(mmu, reg)
		cpu.SetFlagZNHC(val == 0, false, halfCarry, cpu.FlagC())
		cpu.IncPC(1)

	case (opLow%8 == 5) && (0 <= opHigh && opHigh <= 3): // DEC (B|C|D|E|H|L|(HL)|A)
		reg := opHigh*2 + (opLow-5)/8
		dbgpr("0x%08x: DEC %s", cpu.PC(), reg2str(reg))
		val, halfCarry := cpu.decReg(mmu, reg)
		cpu.SetFlagZNHC(val == 0, true, halfCarry, cpu.FlagC())
		cpu.IncPC(1)

	case opLow == 0x0e && (0 <= opHigh && opHigh <= 3): // LD (C|E|L|A), d8
		reg := opHigh*2 + 1
		dbgpr("0x%08x: LD %s, 0x%x", cpu.PC(), reg2str(reg), imm8)
		cpu.setReg(mmu, reg, imm8)
		cpu.IncPC(2)

	case 0x40 <= opcode && opcode <= 0x7f && opcode != 0x76 /* not HALT */ : // LD reg1,reg2
		reg1 := (opcode & 0x3f) >> 3
		reg2 := (opcode & 0x07)
		dbgpr("0x%08x: LD %s, %s", cpu.PC(), reg2str(reg1), reg2str(reg2))
		val := cpu.getReg(mmu, reg2)
		cpu.setReg(mmu, reg1, val)
		cpu.IncPC(1)

	case 0x80 <= opcode && opcode <= 0xbf:
		reg := (opcode & 0x07)
		val := cpu.getReg(mmu, reg)
		var res uint8 = 0
		n, h, c := false, false, false
		switch {
		case 0x80 <= opcode && opcode <= 0x87: // ADD A, reg
			dbgpr("0x%08x: ADD A, %s", cpu.PC(), reg2str(reg))
			res, c = add8(cpu.A(), val, false)
			_, h = add4(cpu.A(), val, false)
		case 0x88 <= opcode && opcode <= 0x8f: // ADC A, reg
			dbgpr("0x%08x: ADC A, %s", cpu.PC(), reg2str(reg))
			res, c = add8(cpu.A(), val, cpu.FlagC())
			_, h = add4(cpu.A(), val, cpu.FlagC())
		case 0x90 <= opcode && opcode <= 0x97: // SUB reg
			dbgpr("0x%08x: SUB %s", cpu.PC(), reg2str(reg))
			res, c = sub8(cpu.A(), val, false)
			_, h = sub4(cpu.A(), val, false)
			n = true
		case 0x98 <= opcode && opcode <= 0x9f: // SBC A, reg
			dbgpr("0x%08x: SBC A, %s", cpu.PC(), reg2str(reg))
			res, c = sub8(cpu.A(), val, cpu.FlagC())
			_, h = sub4(cpu.A(), val, cpu.FlagC())
			n = true
		case 0xa0 <= opcode && opcode <= 0xa7: // AND reg
			dbgpr("0x%08x: AND %s", cpu.PC(), reg2str(reg))
			res = cpu.A() & val
			h = true
		case 0xa8 <= opcode && opcode <= 0xaf: // XOR reg
			dbgpr("0x%08x: XOR %s", cpu.PC(), reg2str(reg))
			res = cpu.A() ^ val
		case 0xb0 <= opcode && opcode <= 0xb7: // OR reg
			dbgpr("0x%08x: OR %s", cpu.PC(), reg2str(reg))
			res = cpu.A() | val
		case 0xb8 <= opcode && opcode <= 0xbf: // CP reg
			dbgpr("0x%08x: CP %s", cpu.PC(), reg2str(reg))
			res, c = sub8(cpu.A(), val, false)
			_, h = sub4(cpu.A(), val, false)
			res = cpu.A() // restore
			n = true
		}
		cpu.SetA(res)
		cpu.SetFlagZNHC(res == 0, n, h, c)
		cpu.IncPC(1)

	case opcode == 0xea:
		dbgpr("0x%08x: LD (0x%x), A", cpu.PC(), imm16)
		mmu.Set8(imm16, cpu.A())
		cpu.IncPC(3)

	case opcode == 0xfa:
		dbgpr("0x%08x: LD A, (0x%x)", cpu.PC(), imm16)
		cpu.SetA(mmu.Get8(imm16))
		cpu.IncPC(3)

	case opcode == 0xe0:
		dbgpr("0x%08x: LDH (0x%x), A", cpu.PC(), imm8)
		addr := 0xff00 + uint16(imm8)
		mmu.Set8(addr, cpu.A())
		cpu.IncPC(2)

	case opcode == 0xf0:
		dbgpr("0x%08x: LDH A, (0x%x)", cpu.PC(), imm8)
		addr := 0xff00 + uint16(imm8)
		cpu.SetA(mmu.Get8(addr))
		cpu.IncPC(2)

	case opcode == 0xc3:
		dbgpr("0x%08x: JP 0x%x", cpu.PC(), mmu.Get16(cpu.PC()+1))
		addr := mmu.Get16(cpu.PC() + 1)
		cpu.SetPC(addr)

	case opcode == 0xc9:
		dbgpr("0x%08x: RET", cpu.PC())
		addr := cpu.pop16(mmu)
		cpu.SetPC(addr)

	case opcode == 0xcd:
		dbgpr("0x%08x: CALL 0x%x", cpu.PC(), imm16)
		cpu.push16(mmu, cpu.PC()+3)
		cpu.SetPC(imm16)

	case opcode == 0xf3:
		dbgpr("0x%08x: DI", cpu.PC())
		cpu.SetIME(false) // FIXME: Correct? Probably it should be delayed.
		cpu.IncPC(1)

	case opcode == 0xfb:
		dbgpr("0x%08x: EI", cpu.PC())
		cpu.SetIME(true) // FIXME: Correct? Probably it should be delayed.
		cpu.IncPC(1)

	case opLow == 5 && (0xc <= opHigh && opHigh <= 0xf):
		index := opHigh - 0xc
		dbgpr("0x%08x: PUSH %s", cpu.PC(), regBC_DE_HL_AF_ToStr(index))
		cpu.push16(mmu, cpu.getReg16(index, false))
		cpu.IncPC(1)

	case opLow == 1 && (0xc <= opHigh && opHigh <= 0xf):
		index := opHigh - 0xc
		dbgpr("0x%08x: POP  %s", cpu.PC(), regBC_DE_HL_AF_ToStr(index))
		cpu.setReg16(mmu, index, cpu.pop16(mmu), false)
		cpu.IncPC(1)

	default:
		return fmt.Errorf("Illegal instr: 0x%02x at 0x%08x\n", mmu.Get8(cpu.PC()), cpu.PC())
	}

	dbgpr("                af=%04x    bc=%04x    de=%04x    hl=%04x", cpu.AF(), cpu.BC(), cpu.DE(), cpu.HL())
	dbgpr("                sp=%04x    pc=%04x    Z=%d  N=%d  H=%d  C=%d", cpu.SP(), cpu.PC(), b2u8(cpu.FlagZ()), b2u8(cpu.FlagN()), b2u8(cpu.FlagH()), b2u8(cpu.FlagC()))

	return nil
}
