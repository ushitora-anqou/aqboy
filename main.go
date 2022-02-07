package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/veandco/go-sdl2/sdl"

	"github.com/ushitora-anqou/aqboy/bus"
	"github.com/ushitora-anqou/aqboy/cpu"
	"github.com/ushitora-anqou/aqboy/mmu"
	"github.com/ushitora-anqou/aqboy/ppu"
	"github.com/ushitora-anqou/aqboy/timer"
	"github.com/ushitora-anqou/aqboy/util"
)

func buildUsageError() error {
	return fmt.Errorf("Usage: %s PATH [BREAKPOINT-ADDR]", os.Args[0])
}

type SDLWindow struct {
	window        *sdl.Window
	renderer      *sdl.Renderer
	texture       *sdl.Texture
	width, height int32
	srcPic        [ppu.LCD_WIDTH * ppu.LCD_HEIGHT]uint8
}

func NewSDLWindow() (*SDLWindow, error) {
	var width, height int32 = ppu.LCD_WIDTH * 4, ppu.LCD_HEIGHT * 4
	window, err := sdl.CreateWindow(
		"aqboy",
		sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		width,
		height,
		sdl.WINDOW_SHOWN,
	)
	if err != nil {
		return nil, err
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return nil, err
	}

	texture, err := renderer.CreateTexture(
		sdl.PIXELFORMAT_ARGB8888,
		sdl.TEXTUREACCESS_STREAMING,
		ppu.LCD_WIDTH,
		ppu.LCD_HEIGHT,
	)
	if err != nil {
		return nil, err
	}

	return &SDLWindow{
		window,
		renderer,
		texture,
		width,
		height,
		[ppu.LCD_WIDTH * ppu.LCD_HEIGHT]uint8{},
	}, nil
}

func (wind *SDLWindow) DrawLine(ly int, scanline []uint8) error {
	if len(scanline) != ppu.LCD_WIDTH {
		return fmt.Errorf(
			"Invalid length of scanline data: expected %d, got %d",
			ppu.LCD_WIDTH,
			len(scanline),
		)
	}
	copy(wind.srcPic[ly*ppu.LCD_WIDTH:(ly+1)*ppu.LCD_WIDTH], scanline)
	return nil
}

type WindowEvent struct {
	Escape bool
}

func (wind *SDLWindow) HandleEvents() *WindowEvent {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch event.(type) {
		case *sdl.QuitEvent:
			return &WindowEvent{Escape: true}
		case *sdl.KeyboardEvent:
			switch event.(*sdl.KeyboardEvent).Keysym.Sym {
			case sdl.K_ESCAPE:
				return &WindowEvent{Escape: true}
			}
		}
	}
	return &WindowEvent{}
}

func (wind *SDLWindow) UpdateScreen() error {
	// Update the texture
	pixels, _, err := wind.texture.Lock(nil)
	if err != nil {
		return err
	}
	for row := 0; row < ppu.LCD_HEIGHT; row++ {
		for col := 0; col < ppu.LCD_WIDTH; col++ {
			off := row*ppu.LCD_WIDTH + col
			var color byte = 0
			switch wind.srcPic[off] {
			case 0: // White
				color = 0xff
			case 1: // Light gray
				color = 0xcc
			case 2: // Dark gray
				color = 0x44
			case 3: // Black
				color = 0x00
			}
			pixels[off*4+0] = color // b
			pixels[off*4+1] = color // g
			pixels[off*4+2] = color // r
			pixels[off*4+3] = 0xff  // a
		}
	}
	wind.texture.Unlock()

	// Present the scene
	wind.renderer.Clear()
	wind.renderer.Copy(wind.texture, nil, nil)
	wind.renderer.Present()

	return nil
}

type TimeSynchronizer struct {
	prevTime   time.Time
	usPerFrame int
}

func NewTimeSynchronizer(targetFPS int) *TimeSynchronizer {
	return &TimeSynchronizer{
		prevTime:   time.Now(),
		usPerFrame: 1000000 / targetFPS,
	}
}

func (ts *TimeSynchronizer) maySleep() {
	curTime := time.Now()
	dur := curTime.Sub(ts.prevTime)
	diff := ts.usPerFrame - int(dur.Microseconds())
	if diff > 0 {
		time.Sleep(time.Duration(diff) * time.Microsecond)
	}
	ts.prevTime = curTime
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
	if os.Getenv("AQBOY_TRACE") == "1" {
		util.EnableTrace()
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

	// Initialize SDL
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		log.Fatal(err)
	}
	defer sdl.Quit()
	// Create a window
	wind, err := NewSDLWindow()
	if err != nil {
		log.Fatal(err)
	}

	// Build up the bus
	bus.Register(cpu, mmu, ppu, wind, timer)

	// Main loop
	synchronizer := NewTimeSynchronizer(60 /* FPS */)
LabelMainLoop:
	for {
		// Handle inputs/events
		event := wind.HandleEvents()
		if event.Escape {
			break
		}

		// Compute
		for cnt := 0; cnt < 145*(144+10); { // Emulate one frame
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
		synchronizer.maySleep()
	}
	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
