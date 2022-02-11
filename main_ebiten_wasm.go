//go:build ebiten && wasm

package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ushitora-anqou/aqboy/constant"
	"github.com/ushitora-anqou/aqboy/util"
	"github.com/ushitora-anqou/aqboy/window"
)

type FooGame struct{}

func (g *FooGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return constant.LCD_WIDTH, constant.LCD_HEIGHT
}

func (g *FooGame) Update() error {
	return nil
}

func (g *FooGame) Draw(screen *ebiten.Image) {
}

type Game struct {
	aqboy *AQBoy
	wind  *window.EbitenWindow
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

	if g.aqboy != nil {
		g.aqboy.Update(event)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	pixels := g.wind.Render()
	screen.ReplacePixels(pixels)
}

func (g *Game) Reset(rom []uint8) error {
	aqboy, err := NewAQBoy(g.wind, rom)
	if err != nil {
		return err
	}
	g.aqboy = aqboy
	return nil
}

var game *Game

func runEbiten() error {
	if err := window.EbitenInitialize(); err != nil {
		return err
	}

	wind, err := window.NewEbitenWindow()
	if err != nil {
		return err
	}

	game = &Game{nil, wind}

	return ebiten.RunGame(game)
}

func startEmulator(this js.Value, args []js.Value) interface{} {
	base64ROM := args[0].String()
	rom, err := base64.StdEncoding.DecodeString(base64ROM)
	if err != nil {
		return nil
	}
	fmt.Printf("ROM size: %d\n", len(rom))

	if game != nil {
		err := game.Reset(rom)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
		}
	}

	fmt.Printf("Reset done\n")

	return nil
}

func main() {
	println("Go WebAssembly Initialized")
	js.Global().Set("aqboyStartEmulator", js.FuncOf(startEmulator))
	runEbiten()
}
