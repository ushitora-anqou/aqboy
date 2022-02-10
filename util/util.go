package util

func BoolToU8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

type TickCounter struct {
	current, target uint
}

func NewTickCounter(target uint) *TickCounter {
	return &TickCounter{target: target}
}

func (tc *TickCounter) Tick(tick uint) bool {
	posedge := false
	tc.current += tick
	if tc.current > tc.target {
		tc.current -= tc.target
		posedge = true
	}
	return posedge
}
