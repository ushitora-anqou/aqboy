package main

import (
	"log"
	"os"
	"strconv"

	"github.com/ushitora-anqou/aqboy/cpu"
	"github.com/ushitora-anqou/aqboy/mmu"
)

func run() error {
	cpu := cpu.NewCPU()
	//mmu, err := mmu.NewMMU("misc/cpu_instrs/individual/01-special.gb")
	mmu, err := mmu.NewMMU("misc/cpu_instrs/individual/03-op sp,hl.gb")
	if err != nil {
		return err
	}

	var breakpointAddr uint16 = 0
	if len(os.Args) >= 2 {
		addr, err := strconv.ParseUint(os.Args[1], 0, 16)
		if err != nil {
			return err
		}
		breakpointAddr = uint16(addr)
	}
	for {
		if breakpointAddr == cpu.PC() {
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
