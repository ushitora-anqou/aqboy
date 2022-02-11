// +build USE_SDL2

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	"github.com/ushitora-anqou/aqboy/window"
)

func runSDL2() error {
	// Parse options and arguments
	flag.Parse()
	if flag.NArg() < 1 {
		return fmt.Errorf("Usage: %s PATH", os.Args[0])
	}
	romPath := flag.Arg(0)
	if filename := os.Getenv("AQBOY_CPUPROFILE"); filename != "" {
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
		if err := pprof.StartCPUProfile(file); err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
	}

	// Initialize SDL
	if err := window.SDLInitialize(); err != nil {
		return err
	}

	// Create a window
	wind, err := window.NewSDLWindow()
	if err != nil {
		return err
	}

	// Go emulation
	aqboy, err := NewAQBoy(wind, romPath)
	if err != nil {
		return err
	}
	return aqboy.Run()
}

func main() {
	err := runSDL2()
	if err != nil {
		log.Fatal(err)
	}
}
