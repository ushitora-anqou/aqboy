//go:build ebiten && !wasm

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ushitora-anqou/aqboy/constant"
	"github.com/ushitora-anqou/aqboy/util"
	"github.com/ushitora-anqou/aqboy/window"
)

type Game struct {
	aqboy *AQBoy
	wind  *window.EbitenWindow
}

func NewGame(wind *window.EbitenWindow, aqboy *AQBoy) (*Game, error) {
	game := &Game{
		aqboy,
		wind,
	}
	return game, nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return constant.LCD_WIDTH, constant.LCD_HEIGHT
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		os.Exit(0)
	}

	event := &window.WindowEvent{}
	event.Direction |= util.BoolToU8(ebiten.IsKeyPressed(ebiten.KeyW)) << constant.DIR_UP
	event.Direction |= util.BoolToU8(ebiten.IsKeyPressed(ebiten.KeyA)) << constant.DIR_LEFT
	event.Direction |= util.BoolToU8(ebiten.IsKeyPressed(ebiten.KeyD)) << constant.DIR_RIGHT
	event.Direction |= util.BoolToU8(ebiten.IsKeyPressed(ebiten.KeyS)) << constant.DIR_DOWN
	event.Action |= util.BoolToU8(ebiten.IsKeyPressed(ebiten.KeyK)) << constant.ACT_A
	event.Action |= util.BoolToU8(ebiten.IsKeyPressed(ebiten.KeyJ)) << constant.ACT_B
	event.Action |= util.BoolToU8(ebiten.IsKeyPressed(ebiten.KeyEnter)) << constant.ACT_START
	event.Action |= util.BoolToU8(ebiten.IsKeyPressed(ebiten.KeySpace)) << constant.ACT_SELECT

	g.aqboy.Update(event)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	pixels := g.wind.Render()
	screen.ReplacePixels(pixels)
}

func runEbiten(rom []uint8) error {
	if err := window.EbitenInitialize(); err != nil {
		return err
	}

	wind, err := window.NewEbitenWindow()
	if err != nil {
		return err
	}

	aqboy, err := NewAQBoy(wind, rom)
	if err != nil {
		return err
	}

	game, err := NewGame(wind, aqboy)
	if err != nil {
		return err
	}

	return ebiten.RunGame(game)
}

func run() error {
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

	rom, err := os.ReadFile(romPath)
	if err != nil {
		return err
	}

	return runEbiten(rom)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
