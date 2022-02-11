//go:build ebiten

package window

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/ushitora-anqou/aqboy/constant"
)

var palette = [4]uint8{
	constant.COLOR_WHITE,
	constant.COLOR_LIGHT_GRAY,
	constant.COLOR_DARK_GRAY,
	constant.COLOR_BLACK,
}

func EbitenInitialize() error {
	ebiten.SetMaxTPS(60)
	ebiten.SetWindowSize(constant.WINDOW_WIDTH, constant.WINDOW_HEIGHT)
	ebiten.SetWindowTitle(constant.WINDOW_TITLE)

	audio.NewContext(constant.AUDIO_FREQ)

	return nil
}

type EbitenWindow struct {
	srcPic         [constant.LCD_WIDTH * constant.LCD_HEIGHT]uint8
	audioPlayer    *audio.Player
	audioBuffer    [][]uint8
	mtxAudioBuffer sync.Mutex
}

func NewEbitenWindow() (*EbitenWindow, error) {
	if constant.CHANNELS != 2 {
		return nil, fmt.Errorf("Invalid channel: ebiten supports only 2 channels.")
	}

	wind := &EbitenWindow{}
	player, err := audio.CurrentContext().NewPlayer(NewEbitenAudioReader(wind))
	if err != nil {
		return nil, err
	}
	player.Play()
	wind.audioPlayer = player
	return wind, nil
}

func (wind *EbitenWindow) DrawLine(ly int, scanline []uint8) error {
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

func (wind *EbitenWindow) Render() []uint8 {
	pixels := make([]uint8, 4*constant.LCD_WIDTH*constant.LCD_HEIGHT)
	for row := 0; row < constant.LCD_HEIGHT; row++ {
		for col := 0; col < constant.LCD_WIDTH; col++ {
			off := row*constant.LCD_WIDTH + col
			color := palette[wind.srcPic[off]]
			pixels[off*4+0] = color // r
			pixels[off*4+1] = color // g
			pixels[off*4+2] = color // b
			pixels[off*4+3] = 0xff  // a
		}
	}
	return pixels
}

func (wind *EbitenWindow) EnqueueAudioBuffer(buf []float32) error {
	length := constant.AUDIO_SAMPLES * constant.CHANNELS
	if len(buf) != length {
		return fmt.Errorf("Invalid length of audio buffer")
	}

	bufU := make([]uint8, length*2 /* 16 bits */)
	for i := 0; i < length; i++ {
		// signed, 16-bit, and little endian
		val := int16(buf[i] * 0x7fff)
		bufU[i*2] = uint8(val & 0x00ff)
		bufU[i*2+1] = uint8((val >> 8) & 0x00ff)
	}

	wind.mtxAudioBuffer.Lock()
	defer wind.mtxAudioBuffer.Unlock()

	if len(wind.audioBuffer) >= constant.AUDIO_QUEUE_SIZE {
		wind.audioBuffer = wind.audioBuffer[1:] // Discard the old one
	}

	// Enqueue
	wind.audioBuffer = append(wind.audioBuffer, bufU)

	return nil
}

type EbitenAudioReader struct {
	wind *EbitenWindow
}

func NewEbitenAudioReader(wind *EbitenWindow) *EbitenAudioReader {
	return &EbitenAudioReader{wind}
}

func (r *EbitenAudioReader) Read(buf []uint8) (int, error) {
	wind := r.wind
	wind.mtxAudioBuffer.Lock()

	if len(wind.audioBuffer) == 0 {
		wind.mtxAudioBuffer.Unlock()

		// Return no sound
		length := constant.AUDIO_SAMPLES * constant.CHANNELS * 2 // 16 bits
		if len(buf) < length {
			length = len(buf)
		}
		for i := 0; i < length/2; i++ {
			// No sound in 16-bit little endian
			buf[i*2] = 0
			buf[i*2+1] = 0xff
		}
		return length, nil
	}

	defer wind.mtxAudioBuffer.Unlock()

	src := wind.audioBuffer[0]
	length := copy(buf, src)
	if length == len(src) {
		wind.audioBuffer = wind.audioBuffer[1:]
	} else {
		copy(src, src[length:])
		wind.audioBuffer[0] = src[:len(src)-length]
	}

	return length, nil
}
