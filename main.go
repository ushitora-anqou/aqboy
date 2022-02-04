package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/veandco/go-sdl2/sdl"

	"github.com/ushitora-anqou/aqboy/cpu"
	"github.com/ushitora-anqou/aqboy/mmu"
)

func buildUsageError() error {
	return fmt.Errorf("Usage: %s PATH [BREAKPOINT-ADDR]", os.Args[0])
}

type AtomicBool struct {
	flag uint32
}

func NewAtomicBool(initVal bool) *AtomicBool {
	if initVal {
		return &AtomicBool{flag: 1}
	} else {
		return &AtomicBool{flag: 0}
	}
}

func (b *AtomicBool) Get() bool {
	return atomic.LoadUint32(&b.flag) != 0
}

func (b *AtomicBool) Set(newVal bool) {
	if newVal {
		atomic.StoreUint32(&b.flag, 1)
	} else {
		atomic.StoreUint32(&b.flag, 0)
	}
}

type Window interface {
	updateVRAM(newVRAM []uint8) error
}

type SDLWindow struct {
	window  *sdl.Window
	surface *sdl.Surface
}

func NewSDLWindow() (*SDLWindow, error) {
	window, err := sdl.CreateWindow(
		"aqboy",
		sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		800,
		600,
		sdl.WINDOW_SHOWN,
	)
	if err != nil {
		return nil, err
	}

	surface, err := window.GetSurface()
	if err != nil {
		return nil, err
	}
	surface.FillRect(nil, 0)
	window.UpdateSurface()

	return &SDLWindow{
		window,
		surface,
	}, nil
}

func (wind *SDLWindow) updateVRAM(newVRAM []uint8) error {
	return nil
}

func run() error {
	// Parse options and arguments
	flag.Parse()
	if flag.NArg() < 1 {
		return buildUsageError()
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

	// Initialize SDL
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		log.Fatal(err)
	}
	defer sdl.Quit()
	// Create a window
	_, err := NewSDLWindow()
	if err != nil {
		log.Fatal(err)
	}

	// Build a new CPU
	cpu := cpu.NewCPU()
	// Load ROM
	mmu, err := mmu.NewMMU(romPath)
	if err != nil {
		return err
	}

	// Start computing
	running := NewAtomicBool(true)
	go func() {
		for running.Get() {
			if breakpointAddr != nil && *breakpointAddr == cpu.PC() {
				break
			}
			err := cpu.Step(mmu)
			if err != nil {
				running.Set(false)
				log.Fatal(err)
			}
		}
		running.Set(false)
	}()

	// Start Drawing
	for running.Get() {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				running.Set(false)
			}
		}
	}

	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
