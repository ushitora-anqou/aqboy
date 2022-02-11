//go:build sdl2

package window

// typedef float Float32;
// typedef unsigned char Uint8;
// void OnAudioPlayback(void *userdata, Uint8 *stream, int len);
import "C"
import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/mattn/go-pointer"
	"github.com/ushitora-anqou/aqboy/constant"
	"github.com/veandco/go-sdl2/sdl"
)

var palette = [4]uint8{
	constant.COLOR_WHITE,
	constant.COLOR_LIGHT_GRAY,
	constant.COLOR_DARK_GRAY,
	constant.COLOR_BLACK,
}

func SDLInitialize() error {
	return sdl.Init(sdl.INIT_EVERYTHING)
}

type SDLWindow struct {
	window                    *sdl.Window
	renderer                  *sdl.Renderer
	texture                   *sdl.Texture
	srcPic                    [constant.LCD_WIDTH * constant.LCD_HEIGHT]uint8
	prevAction, prevDirection uint8
	audioDevice               sdl.AudioDeviceID
	audioBuffer               [][]C.Float32 // NOTE: Access to this variable must be mutually excluded by sdl.LockAudioDevice(audioDevice).
}

func NewSDLWindow() (*SDLWindow, error) {
	window, err := sdl.CreateWindow(
		constant.WINDOW_TITLE,
		sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		constant.WINDOW_WIDTH,
		constant.WINDOW_HEIGHT,
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
		constant.LCD_WIDTH,
		constant.LCD_HEIGHT,
	)
	if err != nil {
		return nil, err
	}

	wind := &SDLWindow{
		window:      window,
		renderer:    renderer,
		texture:     texture,
		srcPic:      [constant.LCD_WIDTH * constant.LCD_HEIGHT]uint8{},
		audioBuffer: [][]C.Float32{},
	}

	audioDevice, err := sdl.OpenAudioDevice(
		"",
		false,
		&sdl.AudioSpec{
			Freq:     constant.AUDIO_FREQ,
			Format:   sdl.AUDIO_F32,
			Channels: constant.CHANNELS,
			Samples:  constant.AUDIO_SAMPLES,
			Callback: sdl.AudioCallback(C.OnAudioPlayback),
			UserData: pointer.Save(wind),
		},
		nil,
		0,
	)
	if err != nil {
		return nil, err
	}
	sdl.PauseAudioDevice(audioDevice, false)
	wind.audioDevice = audioDevice

	return wind, nil
}

func (wind *SDLWindow) DrawLine(ly int, scanline []uint8) error {
	if len(scanline) != constant.LCD_WIDTH {
		return fmt.Errorf(
			"Invalid length of scanline data: expected %d, got %d",
			constant.LCD_WIDTH,
			len(scanline),
		)
	}
	copy(wind.srcPic[ly*constant.LCD_WIDTH:(ly+1)*constant.LCD_WIDTH], scanline)
	return nil
}

func (wind *SDLWindow) HandleEvents() (bool, *WindowEvent) {
	we := &WindowEvent{
		Action:    wind.prevAction,
		Direction: wind.prevDirection,
	}
	escape := false

	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch event.(type) {
		case *sdl.QuitEvent:
			escape = true
			break

		case *sdl.KeyboardEvent:
			kbEvent := event.(*sdl.KeyboardEvent)
			switch kbEvent.Type {
			case sdl.KEYDOWN:
				switch kbEvent.Keysym.Sym {
				case sdl.K_ESCAPE:
					escape = true
				case sdl.K_w:
					we.Direction |= (1 << constant.DIR_UP)
				case sdl.K_a:
					we.Direction |= (1 << constant.DIR_LEFT)
				case sdl.K_d:
					we.Direction |= (1 << constant.DIR_RIGHT)
				case sdl.K_s:
					we.Direction |= (1 << constant.DIR_DOWN)
				case sdl.K_k:
					we.Action |= (1 << constant.ACT_A)
				case sdl.K_j:
					we.Action |= (1 << constant.ACT_B)
				case sdl.K_RETURN:
					we.Action |= (1 << constant.ACT_START)
				case sdl.K_SPACE:
					we.Action |= (1 << constant.ACT_SELECT)
				}

			case sdl.KEYUP:
				switch kbEvent.Keysym.Sym {
				case sdl.K_w:
					we.Direction &^= (1 << constant.DIR_UP)
				case sdl.K_a:
					we.Direction &^= (1 << constant.DIR_LEFT)
				case sdl.K_d:
					we.Direction &^= (1 << constant.DIR_RIGHT)
				case sdl.K_s:
					we.Direction &^= (1 << constant.DIR_DOWN)
				case sdl.K_k:
					we.Action &^= (1 << constant.ACT_A)
				case sdl.K_j:
					we.Action &^= (1 << constant.ACT_B)
				case sdl.K_RETURN:
					we.Action &^= (1 << constant.ACT_START)
				case sdl.K_SPACE:
					we.Action &^= (1 << constant.ACT_SELECT)
				}
			}
		}
	}

	wind.prevAction = we.Action
	wind.prevDirection = we.Direction

	return escape, we
}

func (wind *SDLWindow) UpdateScreen() error {
	// Update the texture
	pixels, _, err := wind.texture.Lock(nil)
	if err != nil {
		return err
	}
	for row := 0; row < constant.LCD_HEIGHT; row++ {
		for col := 0; col < constant.LCD_WIDTH; col++ {
			off := row*constant.LCD_WIDTH + col
			color := palette[wind.srcPic[off]]
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

func (wind *SDLWindow) EnqueueAudioBuffer(buf []float32) error {
	// Lock the device to avoid data race with OnAudioPlayback.
	sdl.LockAudioDevice(wind.audioDevice)
	defer sdl.UnlockAudioDevice(wind.audioDevice)

	length := constant.AUDIO_SAMPLES * constant.CHANNELS
	if len(buf) != length {
		return fmt.Errorf("Invalid length of audio buffer")
	}

	if len(wind.audioBuffer) >= constant.AUDIO_QUEUE_SIZE {
		wind.popAudioBuffer() // Discard the old one
	}

	// Copy buf to bufC.
	// FIXME: Maybe there is faster technique.
	bufC := make([]C.Float32, length)
	for i, v := range buf {
		bufC[i] = C.Float32(v)
	}

	// Enqueue
	wind.audioBuffer = append(wind.audioBuffer, bufC)

	return nil
}

// popAudioBuffer assumes that access to wind.audioBuffer is locked beforehand.
func (wind *SDLWindow) popAudioBuffer() []C.Float32 {
	if len(wind.audioBuffer) == 0 {
		return nil
	}

	ret := wind.audioBuffer[0]
	wind.audioBuffer = wind.audioBuffer[1:]
	return ret
}

//export OnAudioPlayback
func OnAudioPlayback(userdata unsafe.Pointer, stream *C.Uint8, length C.int) {
	n := int(length) / 4
	hdr := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(stream)), Len: n, Cap: n}
	buf := *(*[]C.Float32)(unsafe.Pointer(&hdr))
	wind := pointer.Restore(userdata).(*SDLWindow)
	src := wind.popAudioBuffer()

	if src == nil {
		for i := range buf {
			buf[i] = 0
		}
	} else {
		copy(buf, src)
	}
}

type SDLTimeSynchronizer struct {
	prevTicks, usPerFrame int64
}

func NewSDLTimeSynchronizer(targetFPS float64) *SDLTimeSynchronizer {
	return &SDLTimeSynchronizer{
		prevTicks:  int64(sdl.GetTicks()) * 1000,
		usPerFrame: int64(1000000.0 / targetFPS),
	}
}

func (ts *SDLTimeSynchronizer) MaySleep() {
	cur := int64(sdl.GetTicks()) * 1000
	if cur < ts.prevTicks {
		return
	}
	diff := ts.usPerFrame - (cur - ts.prevTicks)
	if diff > 1000 { // Larger than 1ms
		sdl.Delay(uint32(diff / 1000))
	}
	ts.prevTicks += ts.usPerFrame
}
