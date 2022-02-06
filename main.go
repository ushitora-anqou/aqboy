package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/veandco/go-sdl2/sdl"

	"github.com/ushitora-anqou/aqboy/bus"
	"github.com/ushitora-anqou/aqboy/cpu"
	"github.com/ushitora-anqou/aqboy/mmu"
	"github.com/ushitora-anqou/aqboy/ppu"
	"github.com/ushitora-anqou/aqboy/timer"
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

type SDLWindow struct {
	window        *sdl.Window
	renderer      *sdl.Renderer
	texture       *sdl.Texture
	width, height int32
	srcPic        [ppu.LCD_WIDTH * ppu.LCD_HEIGHT]uint8
	mtxSrcPic     sync.Mutex
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
		sync.Mutex{},
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
	wind.mtxSrcPic.Lock()
	copy(wind.srcPic[ly*ppu.LCD_WIDTH:(ly+1)*ppu.LCD_WIDTH], scanline)
	wind.mtxSrcPic.Unlock()
	return nil
}

func (wind *SDLWindow) Update() (bool, error) {
	// Handle inputs
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch event.(type) {
		case *sdl.QuitEvent:
			return false, nil
		case *sdl.KeyboardEvent:
			switch event.(*sdl.KeyboardEvent).Keysym.Sym {
			case sdl.K_ESCAPE:
				return false, nil
			}
		}
	}

	// Update the texture
	pixels, _, err := wind.texture.Lock(nil)
	if err != nil {
		return false, err
	}
	wind.mtxSrcPic.Lock()
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
	wind.mtxSrcPic.Unlock()
	wind.texture.Unlock()

	// Present the scene
	wind.renderer.Clear()
	wind.renderer.Copy(wind.texture, nil, nil)
	wind.renderer.Present()

	return true, nil
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
	//var breakpointAddr *uint16 = nil
	//if flag.NArg() >= 2 {
	//	addr, err := strconv.ParseUint(flag.Arg(1), 0, 16)
	//	if err != nil {
	//		return err
	//	}
	//	addru16 := uint16(addr)
	//	breakpointAddr = &addru16
	//}
	cpu.TRACE_INSTR = os.Getenv("AQBOY_TRACE") == "1"

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
	for {
		// FIXME: Input

		// Compute
		for cnt := 0; cnt < 145*(144+10); { // Emulate one frame
			tick, err := cpu.Step()
			if err != nil {
				return err
			}
			ppu.Update(tick)
			timer.Update(tick)
			cnt += int(tick)
		}

		// Draw
		res, err := wind.Update()
		if err != nil {
			return err
		}
		if !res {
			break
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
