package apu

import (
	"log"

	"github.com/ushitora-anqou/aqboy/util"
)

type channel1 struct {
	sweep, wavePatternDuty, soundLength, volumeEnvelope, freq int
}

type channel2 struct {
	wavePatternDuty, soundLength, volumeEnvelope, freq int
}

type channel3 struct {
	enabled                        bool
	soundLength, outputLevel, freq int
	wave                           [16]uint8
}

type channel4 struct {
	soundLength, volumeEnvelope, polyCounter int
}

type APU struct {
	enabled                                        bool
	so1OutputLevel, so2OutputLevel, outputTerminal int
	ch1                                            channel1
	ch2                                            channel2
	ch3                                            channel3
	ch4                                            channel4
}

func NewAPU() *APU {
	return &APU{}
}

func (apu *APU) Set8(addr uint16, valu8 uint8) {
	val := int(valu8)

	switch addr {
	// Channel 1
	case 0xff10: // NR10
		util.Trace1("\t<<<WRITE: NR10 Channel 1 Sweep register: %08b>>>", val)
		apu.ch1.sweep = val
		return
	case 0xff11: // NR11
		util.Trace1("\t<<<WRITE: NR11 Channel 1 Sound length/Wave pattern duty: %08b>>>", val)
		apu.ch1.wavePatternDuty = val >> 6
		apu.ch1.soundLength = val & 0x3f
		return
	case 0xff12: // NR12
		util.Trace1("\t<<<WRITE: NR12 Channel 1 Volume Envelope: %08b>>>", val)
		apu.ch1.volumeEnvelope = val
		return
	case 0xff13: // NR13
		util.Trace1("\t<<<WRITE: NR13 Channel 1 Frequency lo: %08b>>>", val)
		apu.ch1.freq = (apu.ch1.freq &^ 0xff) | val
		return
	case 0xff14: // NR14
		util.Trace1("\t<<<WRITE: NR14 Channel 1 Frequency hi: %08b>>>", val)
		apu.ch1.freq = (apu.ch1.freq & 0xff) | ((val & 7) << 8)
		// FIXME bit 7 and 6
		return

	// Channel 2
	case 0xff16: // NR21
		util.Trace1("\t<<<WRITE: NR21 Channel 2 Sound Length/Wave Pattern Duty: %08b>>>", val)
		apu.ch2.wavePatternDuty = val >> 6
		apu.ch2.soundLength = val & 0x3f
		return
	case 0xff17: // NR22
		util.Trace1("\t<<<WRITE: NR22 Channel 2 Volume Envelope: %08b>>>", val)
		apu.ch2.volumeEnvelope = val
		return
	case 0xff18: // NR23
		util.Trace1("\t<<<WRITE: NR23 Channel 2 Frequency lo data: %08b>>>", val)
		apu.ch2.freq = (apu.ch2.freq &^ 0xff) | val
		return
	case 0xff19: // NR24
		util.Trace1("\t<<<WRITE: NR23 Channel 2 Frequency hi data: %08b>>>", val)
		apu.ch2.freq = (apu.ch2.freq & 0xff) | ((val & 7) << 8)
		// FIXME bit 7 and 6
		return

	// Channel 3
	case 0xff1a: // NR30
		util.Trace1("\t<<<WRITE: NR30 Channel 3 Sound on/off: %08b>>>", val)
		apu.ch3.enabled = (val >> 7) != 0
		return
	case 0xff1b: // NR31
		util.Trace1("\t<<<WRITE: NR31 Channel 3 Sound Length: 0x%02x>>>", val)
		apu.ch3.soundLength = val
		return
	case 0xff1c: // NR32
		util.Trace1("\t<<<WRITE: NR32 Channel 3 Select output level: %08b>>>", val)
		apu.ch3.outputLevel = val
		return
	case 0xff1d: // NR33
		util.Trace1("\t<<<WRITE: NR33 Channel 3 Frequency's lower data: 0x%02x>>>", val)
		apu.ch3.freq = (apu.ch3.freq &^ 0xff) | val
		return
	case 0xff1e: // NR34
		util.Trace1("\t<<<WRITE: NR34 Channel 3 Frequency's higher data: %08b>>>", val)
		apu.ch3.freq = (apu.ch3.freq & 0xff) | ((val & 7) << 8)
		// FIXME use bit 7 and 6
		return

	// Channel 4
	case 0xff20: // NR41
		util.Trace1("\t<<<WRITE: NR41 Channel 4 Sound Length: 0x%02x>>>", val)
		apu.ch4.soundLength = val
		return
	case 0xff21: // NR42
		util.Trace1("\t<<<WRITE: NR42 Channel 4 Volume Envelope: %08b>>>", val)
		apu.ch4.volumeEnvelope = val
		return
	case 0xff22: // NR43
		util.Trace1("\t<<<WRITE: NR43 Channel 4 Polynomial Counter: %08b>>>", val)
		apu.ch4.polyCounter = val
		return
	case 0xff23: // NR44
		util.Trace1("\t<<<WRITE: NR44 Channel 4 Counter/consecutive; Initial: %08b>>>", val)
		// FIXME: bit7 and bit 6
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
