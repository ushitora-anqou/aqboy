package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strconv"

	"github.com/veandco/go-sdl2/sdl"

	"github.com/ushitora-anqou/aqboy/bus"
	"github.com/ushitora-anqou/aqboy/cpu"
	"github.com/ushitora-anqou/aqboy/mmu"
	"github.com/ushitora-anqou/aqboy/ppu"
	"github.com/ushitora-anqou/aqboy/timer"
	"github.com/ushitora-anqou/aqboy/util"
	"github.com/ushitora-anqou/aqboy/window"
)

func doRun(wind window.Window) error {
	// Parse options and arguments
	flag.Parse()
	if flag.NArg() < 1 {
		return fmt.Errorf("Usage: %s PATH [BREAKPOINT-ADDR]", os.Args[0])
	}
	romPath := flag.Arg(0)
	var breakpointAddr *uint16 = nil
	if flag.NArg() >= 2 {
		addr, err := strconv.ParseUint(flag.Arg(1), 0, 16)
		if err != nil {
			return err
		}
		addru16 := uint16(addr)
		breakpointAddr = &addru16
	}
	if filename := os.Getenv("AQBOY_CPUPROFILE"); filename != "" {
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
		if err := pprof.StartCPUProfile(file); err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
	}

	// Build the components
	bus := bus.NewBus()
	cpu := cpu.NewCPU(bus)
	ppu := ppu.NewPPU(bus)
	mmu, err := mmu.NewMMU(bus, romPath)
	if err != nil {
		return err
	}
	timer := timer.NewTimer(bus)

	// Build up the bus
	bus.Register(cpu, mmu, ppu, wind, timer)

	// Main loop
	synchronizer := window.NewTimeSynchronizer(wind, 60 /* FPS */)
LabelMainLoop:
	for {
		// Handle inputs/events
		event := wind.HandleEvents()
		if event.Escape {
			break
		}

		// Compute
		for cnt := 0; cnt < 456*154; { // Emulate one frame
			tick, err := cpu.Step()
			if err != nil {
				return err
			}
			ppu.Update(tick)
			timer.Update(tick)
			cnt += int(tick)

			util.Trace4("                af=%04x    bc=%04x    de=%04x    hl=%04x", cpu.AF(), cpu.BC(), cpu.DE(), cpu.HL())
			util.Trace6("                sp=%04x    pc=%04x    Z=%d  N=%d  H=%d  C=%d", cpu.SP(), cpu.PC(), util.BoolToU8(cpu.FlagZ()), util.BoolToU8(cpu.FlagN()), util.BoolToU8(cpu.FlagH()), util.BoolToU8(cpu.FlagC()))
			util.Trace2("                ime=%d      tima=%02x", util.BoolToU8(cpu.IME()), timer.TIMA())

			if breakpointAddr != nil && cpu.PC() == *breakpointAddr {
				break LabelMainLoop
			}
		}

		// Draw
		err := wind.UpdateScreen()
		if err != nil {
			return err
		}
		synchronizer.MaySleep()
	}
	return nil
}

func run() error {
	// Initialize SDL
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return err
	}
	defer sdl.Quit()
	// Create a window
	wind, err := window.NewSDLWindow()
	if err != nil {
		return err
	}
	return doRun(wind)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
