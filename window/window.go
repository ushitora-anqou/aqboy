package window

type WindowEvent struct {
	Escape            bool
	Direction, Action uint8
}

type Window interface {
	DrawLine(ly int, scanline []uint8) error
	HandleEvents() *WindowEvent
	UpdateScreen() error
	getTicks() int64
	delay(val int64)
}
