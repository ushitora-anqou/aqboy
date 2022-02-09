package window

type TimeSynchronizer struct {
	prevTicks, usPerFrame int64
	wind                  Window
}

func NewTimeSynchronizer(wind Window, targetFPS float64) *TimeSynchronizer {
	return &TimeSynchronizer{
		prevTicks:  wind.getTicks(),
		usPerFrame: int64(1000000.0 / targetFPS),
		wind:       wind,
	}
}

func (ts *TimeSynchronizer) MaySleep() {
	cur := ts.wind.getTicks()
	if cur < ts.prevTicks {
		return
	}
	diff := ts.usPerFrame - (cur - ts.prevTicks)
	ts.wind.delay(diff)
	ts.prevTicks += ts.usPerFrame
}
