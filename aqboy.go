package main

import (
	"github.com/ushitora-anqou/aqboy/apu"
	"github.com/ushitora-anqou/aqboy/bus"
	"github.com/ushitora-anqou/aqboy/constant"
	"github.com/ushitora-anqou/aqboy/cpu"
	"github.com/ushitora-anqou/aqboy/joypad"
	"github.com/ushitora-anqou/aqboy/mmu"
	"github.com/ushitora-anqou/aqboy/ppu"
	"github.com/ushitora-anqou/aqboy/timer"
	"github.com/ushitora-anqou/aqboy/window"
)

type AQBoy struct {
	bus    *bus.Bus
	cpu    *cpu.CPU
	ppu    *ppu.PPU
	mmu    *mmu.MMU
	timer  *timer.Timer
	apu    *apu.APU
	joypad *joypad.Joypad
	wind   window.Window
	cnt    int
}

func NewAQBoy(wind window.Window, romPath string) (*AQBoy, error) {
	// Build the components
	bus := bus.NewBus()
	cpu := cpu.NewCPU(bus)
	ppu := ppu.NewPPU(bus)
	mmu, err := mmu.NewMMU(bus, romPath)
	if err != nil {
		return nil, err
	}
	timer := timer.NewTimer(bus)
	apu := apu.NewAPU()
	joypad := joypad.NewJoypad()

	// Build up the bus
	bus.Register(cpu, mmu, ppu, wind, timer, apu, joypad)

	return &AQBoy{bus, cpu, ppu, mmu, timer, apu, joypad, wind, 0}, nil
}

func (a *AQBoy) Update(event *window.WindowEvent) error {
	cpu := a.cpu
	ppu := a.ppu
	timer := a.timer
	apu := a.apu
	joypad := a.joypad
	wind := a.wind

	joypad.SetDirection(event.Direction)
	joypad.SetAction(event.Action)

	// Emulate one frame
	for a.cnt < constant.FRAME_TICKS {
		tick, err := cpu.Step()
		if err != nil {
			return err
		}
		ppu.Update(tick)
		timer.Update(tick)
		if apu.Update(tick) {
			err := wind.EnqueueAudioBuffer(apu.GetAudioBuffer())
			if err != nil {
				return err
			}
		}
		a.cnt += int(tick)

		//util.Trace4("                af=%04x    bc=%04x    de=%04x    hl=%04x",
		//	cpu.AF(), cpu.BC(), cpu.DE(), cpu.HL())
		//util.Trace6("                sp=%04x    pc=%04x    Z=%d  N=%d  H=%d  C=%d",
		//	cpu.SP(), cpu.PC(), util.BoolToU8(cpu.FlagZ()), util.BoolToU8(cpu.FlagN()), util.BoolToU8(cpu.FlagH()), util.BoolToU8(cpu.FlagC()))
		//util.Trace2("                ime=%d      tima=%02x",
		//	util.BoolToU8(cpu.IME()), timer.TIMA())
	}
	a.cnt -= constant.FRAME_TICKS

	return nil
}
