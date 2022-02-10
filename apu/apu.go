package apu

import (
	"log"

	"github.com/ushitora-anqou/aqboy/constant"
	"github.com/ushitora-anqou/aqboy/util"
)

type sweep struct {
	sweepPeriod, sweepShift, initFreq, curFreq int
	outEnabled, isDecrementing                 bool
	tick                                       *util.TickCounter
}

func newSweep(freq, val int) *sweep {
	ret := &sweep{
		sweepPeriod:    val >> 4,
		isDecrementing: (val>>3)&1 != 0,
		sweepShift:     val & 7,
		initFreq:       freq,
		outEnabled:     true,
	}
	ret.trigger()
	return ret
}

func (s *sweep) trigger() {
	s.curFreq = s.initFreq
	s.outEnabled = true

	if s.sweepPeriod == 0 && s.sweepShift == 0 {
		s.tick = nil
	} else {
		period := uint(s.sweepPeriod)
		if s.sweepPeriod == 0 {
			period = 8
		}
		s.tick = util.NewTickCounter(period * 8192 * 4)
	}
}

func (s *sweep) isOutEnabled() bool {
	return s.outEnabled
}

func (s *sweep) getCurrentFreq() int {
	return s.curFreq
}

func (s *sweep) doTick(tick uint) bool {
	if s.tick != nil && s.tick.Tick(tick) {
		freq := s.curFreq >> s.sweepShift
		if s.isDecrementing {
			freq = s.curFreq - freq
		} else {
			freq = s.curFreq + freq
		}
		if 0 < freq && freq < 2048 {
			s.curFreq = freq
			return true
		} else {
			s.tick = nil
			s.outEnabled = false
			return false
		}
	}

	return false
}

type envelope struct {
	currentVolume, initVolume, direction, numSweep int
	tick                                           *util.TickCounter
}

func newEnvelope(val int) *envelope {
	ret := &envelope{
		initVolume: val >> 4,
		direction:  (val >> 3) & 1,
		numSweep:   val & 7,
	}
	ret.trigger()
	return ret
}

func (e *envelope) trigger() {
	if e.numSweep == 0 {
		e.tick = nil
		e.currentVolume = 0xf
	} else {
		e.tick = util.NewTickCounter(uint(e.numSweep) * 8192 * 8)
		e.currentVolume = e.initVolume
	}
}

func (e *envelope) doTick(tick uint) {
	if e.tick != nil && e.tick.Tick(tick) {
		if e.direction == 1 && e.currentVolume < 0xf { // Increase
			e.currentVolume += 1
		} else if e.direction == 0 && e.currentVolume > 0x0 { // Decrease
			e.currentVolume -= 1
		}
	}
}

func (e *envelope) getAmplitude(src float32 /* NOTE: -1 to 1 */) float32 {
	if e.initVolume == 0 {
		return 0
	}
	return src * float32(e.currentVolume) / 15
}

type channelQuad struct {
	// Variables written via MMIO
	env                                          *envelope
	sweep                                        *sweep
	sweepCfg, wavePatternDuty, soundLength, freq int

	// Variables for internal process
	waveDutyPos int
	freqTick    *util.TickCounter
}

func (ch *channelQuad) setSweep(val int) {
	ch.sweepCfg = val
	ch.sweep = newSweep(ch.freq, ch.sweepCfg)
}

func (ch *channelQuad) setLengthAndWavePatternDuty(val int) {
	ch.wavePatternDuty = val >> 6
	ch.soundLength = val & 0x3f
}

func (ch *channelQuad) setEnvelope(val int) {
	ch.env = newEnvelope(val)
}

func (ch *channelQuad) setFreqTick(freq int) {
	ch.freqTick = util.NewTickCounter(uint((2048 - freq)) * 4)
}

func (ch *channelQuad) setFreqLow(val int) {
	ch.freq = (ch.freq &^ 0xff) | val
	ch.setFreqTick(ch.freq)
}

func (ch *channelQuad) setFreqHigh(val int) {
	ch.freq = (ch.freq & 0xff) | ((val & 7) << 8)
	ch.setFreqTick(ch.freq)
}

func (ch *channelQuad) trigger() {
	// Start sweep tick
	if ch.sweep != nil {
		ch.sweep.trigger()
	}

	// Start freq tick.
	ch.setFreqTick(ch.freq)

	// Start envelope tick
	if ch.env != nil {
		ch.env.trigger()
	}
}

func (ch *channelQuad) tick(tick uint) {
	if ch.freqTick != nil && ch.freqTick.Tick(tick) {
		ch.waveDutyPos = (ch.waveDutyPos + 1) % 8
	}

	if ch.sweep != nil && ch.sweep.doTick(tick) {
		ch.setFreqTick(ch.sweep.getCurrentFreq())
	}

	if ch.env != nil {
		ch.env.doTick(tick)
	}
}

func (ch *channelQuad) getAmplitude() float32 {
	if ch.sweep != nil && !ch.sweep.isOutEnabled() {
		return 0
	}

	val := []float32{
		-1.0, -1.0, -1.0, -1.0, -1.0, -1.0, -1.0, +1.0, // duty 0
		-1.0, -1.0, -1.0, -1.0, -1.0, -1.0, +1.0, +1.0, // duty 1
		-1.0, -1.0, -1.0, -1.0, +1.0, +1.0, +1.0, +1.0, // duty 2
		+1.0, +1.0, +1.0, +1.0, +1.0, +1.0, -1.0, -1.0, // duty 3
	}[ch.wavePatternDuty*8+ch.waveDutyPos]
	if ch.env != nil {
		val = ch.env.getAmplitude(val)
	}
	return val
}

type channelWave struct {
	enabled                        bool
	soundLength, outputLevel, freq int
	wave                           [16]uint8

	wavePos  int
	freqTick *util.TickCounter
}

func (ch *channelWave) setEnabled(val bool) {
	ch.enabled = val
}

func (ch *channelWave) setSoundLength(val int) {
	ch.soundLength = val
}

func (ch *channelWave) setOutputLevel(val int) {
	ch.outputLevel = (val >> 5) & 3
}

func (ch *channelWave) setFreqTick() {
	ch.freqTick = util.NewTickCounter(uint((2048 - ch.freq)) * 2)
}

func (ch *channelWave) setFreqLow(val int) {
	ch.freq = (ch.freq &^ 0xff) | val
	ch.setFreqTick()
}

func (ch *channelWave) setFreqHigh(val int) {
	ch.freq = (ch.freq & 0xff) | ((val & 7) << 8)
	ch.setFreqTick()
}

func (ch *channelWave) trigger() {
	// Start freq tick
	ch.setFreqTick()
}

func (ch *channelWave) tick(tick uint) {
	if ch.freqTick == nil {
		return
	}
	if ch.freqTick.Tick(tick) {
		ch.wavePos = (ch.wavePos + 1) % 32
	}
}

func (ch *channelWave) getAmplitude() float32 {
	if !ch.enabled {
		return 0
	}

	val := ch.wave[ch.wavePos/2]
	if ch.wavePos%2 == 0 {
		val >>= 4
	} else {
		val &= 0x0f
	}

	// output level
	switch ch.outputLevel {
	case 0:
		val = 0
	case 1:
		// Do nothing
	case 2:
		val >>= 1
	case 3:
		val >>= 2
	}

	// FIXME length

	valF := float32(val)/7.5 - 1.0
	return valF
}

type channelNoise struct {
	// Variables written via MMIO
	env                                   *envelope
	soundLength, shiftAmount, divisorCode int
	widthMode, soundLengthEnabled         bool

	// Variables for internal process
	lengthExpired        bool
	lfsr                 int
	freqTick, lengthTick *util.TickCounter
}

func (ch *channelNoise) setEnvelope(val int) {
	ch.env = newEnvelope(val)
}

func (ch *channelNoise) setFreqTick() {
	divisor := 8
	if ch.divisorCode > 0 {
		divisor = ch.divisorCode << 4
	}
	ch.freqTick = util.NewTickCounter(uint(divisor << ch.shiftAmount))
}

func (ch *channelNoise) trigger() {
	ch.lfsr = 0x7fff
	ch.lengthExpired = false

	// Start freq tick
	ch.setFreqTick()

	// Start length tick
	ch.lengthTick = util.NewTickCounter(uint(64-ch.soundLength) * 8192 * 2)

	// Start envelope tick
	if ch.env != nil {
		ch.env.trigger()
	}
}

func (ch *channelNoise) enableSoundLength(enable bool) {
	ch.soundLengthEnabled = enable
}

func (ch *channelNoise) setSoundLength(val int) {
	ch.soundLength = val & 0x3f
}

func (ch *channelNoise) setPolynomialCounter(val int) {
	ch.shiftAmount = val >> 4
	ch.widthMode = ((val >> 3) & 1) != 0
	ch.divisorCode = val & 7
	ch.setFreqTick()
}

func (ch *channelNoise) tick(tick uint) {
	if ch.lengthExpired {
		return
	}

	if ch.freqTick != nil && ch.freqTick.Tick(tick) {
		tmp := (ch.lfsr & 1) ^ ((ch.lfsr >> 1) & 1)
		ch.lfsr = (ch.lfsr >> 1) | (tmp << 14)
		if ch.widthMode {
			ch.lfsr &^= (1 << 6)
			ch.lfsr |= (tmp << 6)
		}
	}

	if ch.env != nil {
		ch.env.doTick(tick)
	}

	if ch.soundLengthEnabled {
		if ch.lengthTick != nil && ch.lengthTick.Tick(tick) {
			ch.lengthExpired = true
		}
	}
}

func (ch *channelNoise) getAmplitude() float32 {
	if ch.lengthExpired {
		return 0
	}
	val := float32(1&^ch.lfsr)*2 - 1
	if ch.env != nil {
		val = ch.env.getAmplitude(val)
	}
	return val
}

type APU struct {
	enabled                                        bool
	so1OutputLevel, so2OutputLevel, outputTerminal int
	ch1, ch2                                       channelQuad
	ch3                                            channelWave
	ch4                                            channelNoise
	tickSample                                     *util.TickCounter
	buffer                                         []float32
	bufferIndex                                    int
}

func NewAPU() *APU {
	return &APU{
		buffer:     make([]float32, constant.AUDIO_SAMPLES*constant.CHANNELS),
		tickSample: util.NewTickCounter(constant.CPU_FREQ / constant.AUDIO_FREQ),
	}
}

func (apu *APU) Get8(addr uint16) uint8 {
	switch addr {
	case 0xff24:
		util.Trace0("\t<<<READ: NR50 Channel control / On-OFF / Volume>>>")
		return uint8(apu.so1OutputLevel | (apu.so2OutputLevel << 4))
	case 0xff25:
		util.Trace0("\t<<<READ: NR51 Selection of Sound output terminal>>>")
		return uint8(apu.outputTerminal)
	case 0xff26:
		util.Trace0("\t<<<READ: NR52 Sound on/off>>>")
		return util.BoolToU8(apu.enabled) << 7
	}
	log.Fatalf("Invalid memory access of Get8: at 0x%08x", addr)
	return 0
}

func (apu *APU) Set8(addr uint16, valu8 uint8) {
	val := int(valu8)

	switch addr {
	// Channel 1
	case 0xff10: // NR10
		util.Trace1("\t<<<WRITE: NR10 Channel 1 Sweep register: %08b>>>", val)
		apu.ch1.setSweep(val)
		return
	case 0xff11: // NR11
		util.Trace1("\t<<<WRITE: NR11 Channel 1 Sound length/Wave pattern duty: %08b>>>", val)
		apu.ch1.setLengthAndWavePatternDuty(val)
		return
	case 0xff12: // NR12
		util.Trace1("\t<<<WRITE: NR12 Channel 1 Volume Envelope: %08b>>>", val)
		apu.ch1.setEnvelope(val)
		return
	case 0xff13: // NR13
		util.Trace1("\t<<<WRITE: NR13 Channel 1 Frequency lo: %08b>>>", val)
		apu.ch1.setFreqLow(val)
		return
	case 0xff14: // NR14
		util.Trace1("\t<<<WRITE: NR14 Channel 1 Frequency hi: %08b>>>", val)
		apu.ch1.setFreqHigh(val)
		if ((val >> 7) & 1) != 0 {
			apu.ch1.trigger()
		}
		// FIXME bit 6
		return

	// Channel 2
	case 0xff16: // NR21
		util.Trace1("\t<<<WRITE: NR21 Channel 2 Sound Length/Wave Pattern Duty: %08b>>>", val)
		apu.ch2.setLengthAndWavePatternDuty(val)
		return
	case 0xff17: // NR22
		util.Trace1("\t<<<WRITE: NR22 Channel 2 Volume Envelope: %08b>>>", val)
		apu.ch2.setEnvelope(val)
		return
	case 0xff18: // NR23
		util.Trace1("\t<<<WRITE: NR23 Channel 2 Frequency lo data: %08b>>>", val)
		apu.ch2.setFreqLow(val)
		return
	case 0xff19: // NR24
		util.Trace1("\t<<<WRITE: NR23 Channel 2 Frequency hi data: %08b>>>", val)
		apu.ch2.setFreqHigh(val)
		if ((val >> 7) & 1) != 0 {
			apu.ch2.trigger()
		}
		// FIXME bit 6
		return

	// Channel 3
	case 0xff1a: // NR30
		util.Trace1("\t<<<WRITE: NR30 Channel 3 Sound on/off: %08b>>>", val)
		apu.ch3.setEnabled((val >> 7) != 0)
		return
	case 0xff1b: // NR31
		util.Trace1("\t<<<WRITE: NR31 Channel 3 Sound Length: 0x%02x>>>", val)
		apu.ch3.setSoundLength(val)
		return
	case 0xff1c: // NR32
		util.Trace1("\t<<<WRITE: NR32 Channel 3 Select output level: %08b>>>", val)
		apu.ch3.setOutputLevel(val)
		return
	case 0xff1d: // NR33
		util.Trace1("\t<<<WRITE: NR33 Channel 3 Frequency's lower data: 0x%02x>>>", val)
		apu.ch3.setFreqLow(val)
		return
	case 0xff1e: // NR34
		util.Trace1("\t<<<WRITE: NR34 Channel 3 Frequency's higher data: %08b>>>", val)
		apu.ch3.setFreqHigh(val)
		if ((val >> 7) & 1) != 0 {
			apu.ch3.trigger()
		}
		// FIXME use bit 6
		return

	// Channel 4
	case 0xff20: // NR41
		util.Trace1("\t<<<WRITE: NR41 Channel 4 Sound Length: 0x%02x>>>", val)
		apu.ch4.setSoundLength(val)
		return
	case 0xff21: // NR42
		util.Trace1("\t<<<WRITE: NR42 Channel 4 Volume Envelope: %08b>>>", val)
		apu.ch4.setEnvelope(val)
		return
	case 0xff22: // NR43
		util.Trace1("\t<<<WRITE: NR43 Channel 4 Polynomial Counter: %08b>>>", val)
		apu.ch4.setPolynomialCounter(val)
		return
	case 0xff23: // NR44
		util.Trace1("\t<<<WRITE: NR44 Channel 4 Counter/consecutive; Initial: %08b>>>", val)
		apu.ch4.enableSoundLength(((val >> 6) & 1) != 0)
		if ((val >> 7) & 1) != 0 {
			apu.ch4.trigger()
		}
		return

	// Global registers
	case 0xff24: // NR50
		util.Trace1("\t<<<WRITE: NR50 Channel control / On-OFF / Volume: %08b>>>", val)
		apu.so1OutputLevel = val & 7
		apu.so2OutputLevel = (val >> 4) & 7
		// FIXME: Vin support
		return
	case 0xff25: // NR51
		util.Trace1("\t<<<WRITE: NR51 Selection of Sound output terminal: %08b>>>", val)
		apu.outputTerminal = val
		return
	case 0xff26: // NR52
		util.Trace1("\t<<<WRITE: NR52 Sound on/off: %08b>>>", val)
		apu.enabled = (val >> 7) != 0
		return
	}

	switch {
	case 0xff30 <= addr && addr <= 0xff3f: // Wave Pattern RAM
		apu.ch3.wave[addr-0xff30] = valu8
		return
	}

	log.Fatalf("Invalid memory access of Set8: 0x%02x at 0x%08x", val, addr)
}

func (apu *APU) Update(tick uint) bool {
	if !apu.enabled {
		return false
	}

	apu.ch1.tick(tick)
	apu.ch2.tick(tick)
	apu.ch3.tick(tick)
	apu.ch4.tick(tick)

	if apu.tickSample.Tick(tick) {
		ch1 := apu.ch1.getAmplitude()
		ch2 := apu.ch2.getAmplitude()
		ch3 := apu.ch3.getAmplitude()
		ch4 := apu.ch4.getAmplitude()

		var left, right float32
		if (apu.outputTerminal>>0)&1 != 0 {
			right += ch1
		}
		if (apu.outputTerminal>>1)&1 != 0 {
			right += ch2
		}
		if (apu.outputTerminal>>2)&1 != 0 {
			right += ch3
		}
		if (apu.outputTerminal>>3)&1 != 0 {
			right += ch4
		}
		if (apu.outputTerminal>>4)&1 != 0 {
			left += ch1
		}
		if (apu.outputTerminal>>5)&1 != 0 {
			left += ch2
		}
		if (apu.outputTerminal>>6)&1 != 0 {
			left += ch3
		}
		if (apu.outputTerminal>>7)&1 != 0 {
			left += ch4
		}
		right = right * float32(apu.so1OutputLevel) / 7 / 4
		left = left * float32(apu.so2OutputLevel) / 7 / 4

		apu.buffer[apu.bufferIndex] = left
		apu.buffer[apu.bufferIndex+1] = right
		apu.bufferIndex += 2
		if apu.bufferIndex == len(apu.buffer) {
			apu.bufferIndex = 0
			return true
		}
	}

	return false
}

func (apu *APU) GetAudioBuffer() []float32 {
	return apu.buffer
}
