package util

import "log"

var flagEnableTrace bool = false

func EnableTrace() {
	flagEnableTrace = true
}

func DisableTrace() {
	flagEnableTrace = false
}

func Trace(format string, v ...interface{}) {
	if flagEnableTrace {
		log.Printf(format, v...)
	}
}
