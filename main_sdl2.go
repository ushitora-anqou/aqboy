// +build sdl2

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

	// Read the ROM
	rom, err := os.ReadFile(romPath)
	if err != nil {
		return err
	}

	// Build the emulator
	aqboy, err := NewAQBoy(wind, rom)
	if err != nil {
		return err
	}

	// Main loop
	synchronizer := window.NewSDLTimeSynchronizer(60 /* FPS */)
	for {
		// Handle inputs/events
		escape, event := wind.HandleEvents()
		if escape {
			break
		}

		// Update the emulator
		aqboy.Update(event)

		// Draw
		err := wind.UpdateScreen()
		if err != nil {
			return err
		}
		synchronizer.MaySleep()
	}

	return nil
}

func main() {
	err := runSDL2()
	if err != nil {
		log.Fatal(err)
	}
}
