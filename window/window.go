package window

type WindowEvent struct {
	Direction, Action uint8
}

type Window interface {
	DrawLine(ly int, scanline []uint8) error
	EnqueueAudioBuffer(buf []float32) error
}
