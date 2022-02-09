package window

import (
	"fmt"

	"github.com/ushitora-anqou/aqboy/ppu"
	"github.com/veandco/go-sdl2/sdl"
)

type SDLWindow struct {
	window        *sdl.Window
	renderer      *sdl.Renderer
	texture       *sdl.Texture
	width, height int32
	srcPic        [ppu.LCD_WIDTH * ppu.LCD_HEIGHT]uint8
}

func NewSDLWindow() (*SDLWindow, error) {
	var width, height int32 = ppu.LCD_WIDTH * 5, ppu.LCD_HEIGHT * 5
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

func (wind *SDLWindow) getTicks() int64 {
	return int64(sdl.GetTicks()) * 1000
}

func (wind *SDLWindow) delay(val int64) {
	if val > 1000 { // Larger than 1ms
		sdl.Delay(uint32(val / 1000))
	}
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
