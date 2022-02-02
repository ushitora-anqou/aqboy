package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/ushitora-anqou/aqboy/cpu"
	"github.com/ushitora-anqou/aqboy/mmu"
)

func buildUsageError() error {
	return fmt.Errorf("Usage: %s PATH [BREAKPOINT-ADDR]", os.Args[0])
}

func run() error {
	// Parse options and arguments
	flag.Parse()
	if flag.NArg() < 1 {
		return buildUsageError()
	}
	romPath := flag.Arg(0)
	var breakpointAddr *uint16 = nil
	if flag.NArg() >= 2 {
		addr, err := strconv.ParseUint(flag.Arg(1), 0, 16)
		if err != nil {
			return err
		}
		addru16 := uint16(addr)
		breakpointAddr = &addru16
	}

	// Build a new CPU
	cpu := cpu.NewCPU()
	// Load ROM
	mmu, err := mmu.NewMMU(romPath)
	if err != nil {
		return err
	}

	// Go compute
	for {
		if breakpointAddr != nil && *breakpointAddr == cpu.PC() {
			break
		}
		err := cpu.Step(mmu)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
